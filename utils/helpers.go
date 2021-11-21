package utils

func MergeStringStringMaps(A, B map[string]string) map[string]string {
	for k, v := range B {
		A[k] = v
	}
	return A
}
