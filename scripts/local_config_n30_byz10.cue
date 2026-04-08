package config

config: {
	replicaHosts: ["localhost"]
	clientHosts:  ["localhost"]
	replicas: 30
	clients:  1

	byzantineStrategy: {
		silentproposer: [2, 12, 22]
	}
}
