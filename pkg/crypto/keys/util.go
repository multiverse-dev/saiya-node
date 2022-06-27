package keys

func GetMajorityHonestNodeCount(n int) int {
	return n - (n-1)/2
}

func GetDefaultHonestNodeCount(n int) int {
	return n - (n-1)/3
}
