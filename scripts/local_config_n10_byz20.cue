package config

config: {
	replicaHosts: ["localhost"]
	clientHosts:  ["localhost"]
	replicas: 10
	clients:  1

	byzantineStrategy: {
		silentproposer: [2, 7]
	}
}
