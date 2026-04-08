package config

config: {
	replicaHosts: ["localhost"]
	clientHosts:  ["localhost"]
	replicas: 30
	clients:  1

	byzantineStrategy: {
		increaseview: [2, 5, 8, 12, 15, 18, 22, 25, 28]
	}
}
