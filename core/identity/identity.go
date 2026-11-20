package identity

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/anyproto/any-sync/app"
	"github.com/anyproto/any-sync/util/crypto"
	"go.uber.org/zap"

	"github.com/anyproto/anytype-heart/core/anytype/account"
	"github.com/anyproto/anytype-heart/core/domain"
	"github.com/anyproto/anytype-heart/core/files/fileacl"
	"github.com/anyproto/anytype-heart/core/participants"
	"github.com/anyproto/anytype-heart/pkg/lib/datastore/anystoreprovider"
	"github.com/anyproto/anytype-heart/pkg/lib/localstore/objectstore"
	"github.com/anyproto/anytype-heart/pkg/lib/logging"
	"github.com/anyproto/anytype-heart/pkg/lib/pb/model"
	"github.com/anyproto/anytype-heart/space"
	"github.com/anyproto/anytype-heart/util/keyvaluestore"
)

const CName = "identity"

var log = logging.LoggerNotSugared("anytype-identity")

type Service interface {
	GetMyProfileDetails(ctx context.Context) (identity string, metadataKey crypto.SymKey, details *domain.Details)
	UpdateOwnGlobalName(myIdentityGlobalName string)
	RegisterIdentity(spaceId string, identity string, encryptionKey crypto.SymKey) error
	UnregisterIdentity(spaceId string, identity string)
	UnregisterIdentitiesInSpace(spaceId string)
	WaitProfile(ctx context.Context, identity string) *model.IdentityProfile
	WaitProfileWithKey(ctx context.Context, identity string) (*model.IdentityProfileWithKey, error)
	GetMetadataKey(identity string) (crypto.SymKey, error)
	AddIdentityProfile(identityProfile *model.IdentityProfile, key crypto.SymKey) error
	app.ComponentRunnable
}

type service struct {
	ownProfileSubscription *ownProfileSubscription
	accountService         account.Service
	fileAclService         fileacl.Service

	componentCtx       context.Context
	componentCtxCancel context.CancelFunc

	myIdentity string

	identityProfileCacheStore  keyvaluestore.Store[[]byte]
	identityEncryptionKeyStore keyvaluestore.Store[crypto.SymKey]

	objectStore objectstore.ObjectStore

	lock sync.Mutex
	identityObservers      map[string]map[string]struct{}
	identityEncryptionKeys map[string]crypto.SymKey
	identityProfileCache   map[string]*model.IdentityProfile
}

func New(identityObservePeriod time.Duration, pushIdentityBatchTimeout time.Duration) Service {
	ctx, cancel := context.WithCancel(context.Background())
	return &service{
		componentCtx:           ctx,
		componentCtxCancel:     cancel,
		identityObservers:      make(map[string]map[string]struct{}),
		identityEncryptionKeys: make(map[string]crypto.SymKey),
		identityProfileCache:   make(map[string]*model.IdentityProfile),
	}
}

func (s *service) Init(a *app.App) (err error) {
	s.accountService = app.MustComponent[account.Service](a)
	s.myIdentity = s.accountService.AccountID()
	s.fileAclService = app.MustComponent[fileacl.Service](a)

	objectStore := app.MustComponent[objectstore.ObjectStore](a)
	s.objectStore = objectStore
	spaceService := app.MustComponent[space.Service](a)

	provider := app.MustComponent[anystoreprovider.Provider](a)

	s.identityProfileCacheStore, err = keyvaluestore.New(provider.GetCommonDb(), "identity_profile", keyvaluestore.BytesMarshal, keyvaluestore.BytesUnmarshal)
	if err != nil {
		return fmt.Errorf("init identity profile cache store: %w", err)
	}
	s.identityEncryptionKeyStore, err = keyvaluestore.New(provider.GetCommonDb(), "identity_encryption_keys",
		func(key crypto.SymKey) ([]byte, error) {
			return key.Marshall()
		},
		func(data []byte) (crypto.SymKey, error) {
			return crypto.UnmarshallAESKeyProto(data)
		},
	)
	if err != nil {
		return fmt.Errorf("init identity encryption key store: %w", err)
	}

	s.ownProfileSubscription = newOwnProfileSubscription(
		spaceService, objectStore, s.accountService, s.fileAclService, s,
	)
	return
}

func (s *service) Name() (name string) {
	return CName
}

func (s *service) Run(ctx context.Context) (err error) {
	return s.ownProfileSubscription.run(ctx)
}

func (s *service) Close(ctx context.Context) (err error) {
	s.componentCtxCancel()
	s.ownProfileSubscription.close()
	return nil
}

func (s *service) UpdateOwnGlobalName(myIdentityGlobalName string) {
}

func (s *service) WaitProfile(ctx context.Context, identity string) *model.IdentityProfile {
	profile := s.getProfileFromCache(identity)
	if profile != nil {
		return profile
	}
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-s.componentCtx.Done():
			return nil
		case <-ticker.C:
			profile = s.getProfileFromCache(identity)
			if profile != nil {
				return profile
			}
		}
	}
}

func (s *service) GetMetadataKey(identity string) (crypto.SymKey, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	key, ok := s.identityEncryptionKeys[identity]
	if !ok {
		if s.myIdentity == identity {
			return s.deriveMyEncryptionKey()
		}
		persisted, err := s.identityEncryptionKeyStore.Get(s.componentCtx, identity)
		if err != nil {
			return nil, fmt.Errorf("encryption key for identity does not exist: %w", err)
		}
		s.identityEncryptionKeys[identity] = persisted
		return persisted, nil
	}

	return key, nil
}

func (s *service) WaitProfileWithKey(ctx context.Context, identity string) (*model.IdentityProfileWithKey, error) {
	profile := s.WaitProfile(ctx, identity)
	if profile == nil {
		return nil, fmt.Errorf("wait profile: got nil profile")
	}
	key, err := s.GetMetadataKey(identity)
	if err != nil {
		return nil, err
	}

	keyBytes, err := key.Marshall()
	if err != nil {
		return nil, err
	}

	return &model.IdentityProfileWithKey{
		IdentityProfile: profile,
		RequestMetadata: keyBytes,
	}, nil
}

func (s *service) getProfileFromCache(identity string) *model.IdentityProfile {
	s.lock.Lock()
	defer s.lock.Unlock()
	if profile, ok := s.identityProfileCache[identity]; ok {
		return profile
	}
	return nil
}

func (s *service) indexIconImage(profile *model.IdentityProfile) error {
	if len(profile.IconEncryptionKeys) > 0 {
		return s.fileAclService.StoreFileKeys(domain.FileId(profile.IconCid), profile.IconEncryptionKeys)
	}
	return nil
}

func (s *service) broadcastMyIdentityProfile(identityProfile *model.IdentityProfile) {
	if !s.ownProfileSubscription.isLoaded() {
		return
	}
	s.cacheMyIdentityProfile(identityProfile)
	s.updateParticipants(identityProfile, nil)
}

func (s *service) cacheMyIdentityProfile(identityProfile *model.IdentityProfile) {
	profile := identityProfile
	prepared, err := s.ownProfileSubscription.prepareOwnIdentityProfile()
	if err != nil {
		log.Error("cache own identity profile without the icon keys", zap.Error(err))
	} else {
		profile = prepared
	}

	s.lock.Lock()
	s.identityProfileCache[profile.Identity] = profile
	s.lock.Unlock()
}

func (s *service) refreshMyIdentityProfile() {
	profile, err := s.ownProfileSubscription.prepareOwnIdentityProfile()
	if err != nil {
		log.Error("refresh own identity profile", zap.Error(err))
		return
	}

	s.lock.Lock()
	cached := s.identityProfileCache[profile.Identity]
	if cached != nil && cached.Equal(profile) {
		s.lock.Unlock()
		return
	}
	s.identityProfileCache[profile.Identity] = profile
	s.lock.Unlock()

	s.updateParticipants(profile, nil)
}

func (s *service) AddIdentityProfile(profile *model.IdentityProfile, key crypto.SymKey) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	if _, ok := s.identityEncryptionKeys[profile.Identity]; ok {
		log.Info("addIdentityProfile: profile key already exists, skip", zap.String("identity", profile.Identity))
		return nil
	}

	s.identityEncryptionKeys[profile.Identity] = key
	err := s.identityEncryptionKeyStore.Set(context.Background(), profile.Identity, key)
	if err != nil {
		log.Warn("addIdentityProfile: persist identity encryption key", zap.Error(err))
	}

	err = s.indexIconImage(profile)
	if err != nil {
		log.Error("addIdentityProfile: index icon error", zap.Error(err))
	}

	s.identityProfileCache[profile.Identity] = profile
	return nil
}

func (s *service) updateParticipants(profile *model.IdentityProfile, spaceIds []string) {
	s.lock.Lock()
	if cached, ok := s.identityProfileCache[profile.Identity]; ok {
		profile = cached
	}
	if spaceIds == nil {
		for spaceId := range s.identityObservers[profile.Identity] {
			spaceIds = append(spaceIds, spaceId)
		}
	}
	s.lock.Unlock()
	details := participants.BuildIdentityDetails(profile)
	for _, spaceId := range spaceIds {
		id := domain.NewParticipantId(spaceId, profile.Identity)
		err := participants.ModifyDetails(s.componentCtx, s.objectStore, spaceId, id, nil, details)
		if err != nil {
			log.Error("update participant from identity", zap.String("participantId", id), zap.Error(err))
		}
	}
}

func (s *service) RegisterIdentity(spaceId string, identity string, encryptionKey crypto.SymKey) error {
	s.lock.Lock()
	_, err := s.resolveEncryptionKey(identity, encryptionKey)
	if err != nil {
		s.lock.Unlock()
		return err
	}

	observers := s.identityObservers[identity]
	if observers == nil {
		observers = make(map[string]struct{})
		s.identityObservers[identity] = observers
	}
	observers[spaceId] = struct{}{}

	profile := s.identityProfileCache[identity]
	s.lock.Unlock()

	if profile != nil {
		s.updateParticipants(profile, []string{spaceId})
	}

	return nil
}

func (s *service) resolveEncryptionKey(identity string, encryptionKey crypto.SymKey) (crypto.SymKey, error) {
	if key, ok := s.identityEncryptionKeys[identity]; ok {
		if encryptionKey != nil && !key.Equals(encryptionKey) {
			return nil, fmt.Errorf("encryption key for identity %s already exists and do not match new key", identity)
		}
		return key, nil
	}
	if encryptionKey == nil {
		persisted, err := s.identityEncryptionKeyStore.Get(s.componentCtx, identity)
		if err != nil {
			if identity == s.myIdentity {
				return s.deriveMyEncryptionKey()
			}
			return nil, fmt.Errorf("no encryption key for identity %s: %w", identity, err)
		}
		s.identityEncryptionKeys[identity] = persisted
		return persisted, nil
	}
	s.identityEncryptionKeys[identity] = encryptionKey
	err := s.identityEncryptionKeyStore.Set(s.componentCtx, identity, encryptionKey)
	if err != nil {
		log.Warn("persist identity encryption key", zap.Error(err))
	}
	return encryptionKey, nil
}

func (s *service) deriveMyEncryptionKey() (crypto.SymKey, error) {
	_, key, err := domain.DeriveAccountMetadata(s.accountService.Keys().SignKey)
	if err != nil {
		return nil, fmt.Errorf("derive own metadata key: %w", err)
	}
	s.identityEncryptionKeys[s.myIdentity] = key
	persistErr := s.identityEncryptionKeyStore.Set(s.componentCtx, s.myIdentity, key)
	if persistErr != nil {
		log.Warn("persist own identity encryption key", zap.Error(persistErr))
	}
	return key, nil
}

func (s *service) UnregisterIdentity(spaceId string, identity string) {
	s.lock.Lock()
	defer s.lock.Unlock()

	observers := s.identityObservers[identity]
	if observers == nil {
		return
	}
	delete(observers, spaceId)
}

func (s *service) UnregisterIdentitiesInSpace(spaceId string) {
	s.lock.Lock()
	defer s.lock.Unlock()

	for _, observers := range s.identityObservers {
		delete(observers, spaceId)
	}
}

func (s *service) GetMyProfileDetails(ctx context.Context) (identity string, metadataKey crypto.SymKey, details *domain.Details) {
	return s.ownProfileSubscription.getDetails(ctx)
}
