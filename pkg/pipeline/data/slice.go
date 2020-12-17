package data

func DedupStringSlice(fullSlice []string) []string {
    // create a map of keys from the list of values
    uniqMap := make(map[string]struct{})
    for _, v := range fullSlice {
        uniqMap[v] = struct{}{}
    }

    // turn the map keys into a slice
    uniqSlice := make([]string, 0, len(uniqMap))
    for v := range uniqMap {
        uniqSlice = append(uniqSlice, v)
    }

    return uniqSlice
}
