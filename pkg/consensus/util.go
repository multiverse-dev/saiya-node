package consensus

func GetDefaultHonestNodeCount(n int) int {
	return n - (n-1)/3
}
