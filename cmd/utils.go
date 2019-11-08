package cmd

import "strings"

// To support datasources command

// AssetListContainsCaseInsensitive is for when using user input. It's helpful for the
// cmd line.
func AssetListContainsCaseInsensitive(assetList []string, asset string) bool {
	for _, a := range assetList {
		if strings.EqualFold(asset, a) {
			return true
		}
	}
	return false
}

func AssetListContains(assetList []string, asset string) bool {
	for _, a := range assetList {
		if asset == a {
			return true
		}
	}
	return false
}

func SubtractFromSet(set []string, sub ...string) []string {
	var result []string
	for _, r := range set {
		if !AssetListContains(sub, r) {
			result = append(result, r)
		}
	}
	return result
}

func MergeLists(assets ...[]string) []string {
	acc := []string{}
	for _, list := range assets {
		acc = append(acc, list...)
	}
	return acc
}
