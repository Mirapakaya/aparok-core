package localdiscovery

// selfconnect.go is a no-op for the offline Aparok build: self-connection
// probing is disabled on Android by build tag and not required offline.

func testSelfConnection(ip string) error {
	return nil
}
