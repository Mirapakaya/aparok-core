compile-android-lib:
	gomobile bind -work -tags "$(BUILD_TAG_NETWORK) nogrpcserver gomobile nowatchdog nosigar nomutexdeadlockdetector timetzdata rasterizesvg"  -ldflags "-s -w $(FLAGS)" -v -target=android/arm64 -androidapi 26 -o lib.aar github.com/anyproto/anytype-heart/clientlibrary/service
	tar --exclude ".*" -czvf android_lib_$(VERSION).tar.gz lib.aar protobuf json
	mv android_lib_$(VERSION).tar.gz .release/
