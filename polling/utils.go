// Copyright (c) of parts are held by the various contributors (see the CLA)
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package polling

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/cenkalti/backoff"
	log "github.com/sirupsen/logrus"
)

// Default values for PollingExponentialBackOff.
const (
	DefaultInitialInterval     = 800 * time.Millisecond
	DefaultRandomizationFactor = 0.5
	DefaultMultiplier          = 1.5
	DefaultMaxInterval         = 3 * time.Second
	DefaultMaxElapsedTime      = 10 * time.Second // max 10 seconds
)

// PollingExponentialBackOff creates an instance of ExponentialBackOff
func PollingExponentialBackOff() *backoff.ExponentialBackOff {
	b := &backoff.ExponentialBackOff{
		InitialInterval:     DefaultInitialInterval,
		RandomizationFactor: DefaultRandomizationFactor,
		Multiplier:          DefaultMultiplier,
		MaxInterval:         DefaultMaxInterval,
		MaxElapsedTime:      DefaultMaxElapsedTime,
		Clock:               backoff.SystemClock,
	}
	b.Reset()
	return b
}

func TruncateTo4(v float64) float64 {
	return float64(int64(v*1e4)) / 1e4
}

func TruncateTo8(v float64) float64 {
	return float64(int64(v*1e8)) / 1e8
}

/*          All the assets on pegnet
 *
 *          PegNet,                 PEG,        PEG
 *
 *          US Dollar,              USD,        pUSD
 *          Euro,                   EUR,        pEUR
 *          Japanese Yen,           JPY,        pJPY
 *          Pound Sterling,         GBP,        pGBP
 *          Canadian Dollar,        CAD,        pCAD
 *          Swiss Franc,            CHF,        pCHF
 *          Indian Rupee,           INR,        pINR
 *          Singapore Dollar,       SGD,        pSGD
 *          Chinese Yuan,           CNY,        pCNY
 *          Hong Kong Dollar,       HKD,        pHKD
 * DROPPED  Tiawanese Dollar,       TWD,        pTWD
 *          Korean Won,             KRW,        pKRW
 * DROPPED  Argentine Peso,         ARS,        pARS
 *          Brazil Real,            BRL,        pBRL
 *          Philippine Peso         PHP,        pPHP
 *          Mexican Peso            MXN,        pMXN
 *
 *          Gold Troy Ounce,        XAU,        pXAU
 *          Silver Troy Ounce,      XAG,        pXAG
 *          Palladium Troy Ounce,   XPD,        pXPD
 *          Platinum Troy Ounce,    XPT,        pXPT
 *
 *          Bitcoin,                XBT,        pXBT
 *          Ethereum,               ETH,        pETH
 *          Litecoin,               LTC,        pLTC
 *          Ravencoin,              RVN,        pRVN
 *          Bitcoin Cash,           XBC,        pXBC
 *          Factom,                 FCT,        pFCT
 *          Binance Coin            BNB,        pBNB
 *          Stellar                 XLM,        pXLM
 *          Cardano                 ADA,        pADA
 *          Monero                  XMR,        pXMR
 *          Dash                    DASH,       pDASH
 *          Zcash                   ZEC,      	pZEC
 *          Decred                  DCR,        pDCR
 */

var (
	PEGAsset = []string{
		"PEG",
	}

	CurrencyAssets = []string{
		"USD",
		"EUR",
		"JPY",
		"GBP",
		"CAD",
		"CHF",
		"INR",
		"SGD",
		"CNY",
		"HKD",
		//"TWD",
		"KRW",
		//"ARS",
		"BRL",
		"PHP",
		"MXN",
	}

	CommodityAssets = []string{
		"XAU",
		"XAG",
		"XPD",
		"XPT",
	}

	CryptoAssets = []string{
		"XBT",
		"ETH",
		"LTC",
		"RVN",
		"XBC",
		"FCT",
		"BNB",
		"XLM",
		"ADA",
		"XMR",
		"DASH",
		"ZEC",
		"DCR",
	}

	V4CurrencyAdditions = []string{
		"AUD",
		"NZD",
		"SEK",
		"NOK",
		"RUB",
		"ZAR",
		"TRY",
	}

	V4CryptoAdditions = []string{
		"EOS",
		"LINK",
		"ATOM",
		"BAT",
		"XTZ",
	}

	V4ReferenceAdditions = []string{
		"pUSD",
	}

	AllAssets = MergeLists(PEGAsset, CurrencyAssets, CommodityAssets, CryptoAssets, V4CurrencyAdditions, V4CryptoAdditions, V4ReferenceAdditions)
	AssetsV1  = MergeLists(PEGAsset, CurrencyAssets, CommodityAssets, CryptoAssets)
	// This is with the PNT instead of PEG. Should never be used unless absolutely necessary.
	//
	// Deprecated: Was used for version 1 before PNT -> PEG
	AssetsV1WithPNT = MergeLists([]string{"PNT"}, SubtractFromSet(AssetsV1, "PEG"))
	// Version One, subtract 2 assets
	AssetsV2 = SubtractFromSet(AssetsV1, "XPD", "XPT")

	// Additional assets to V2 set
	AssetsV4 = MergeLists(AssetsV2, V4CurrencyAdditions, V4CryptoAdditions, V4ReferenceAdditions)
)

// AssetListContainsCaseInsensitive is for when using user input. It's helpful for the
// cmd line.
func AssetListContainsCaseInsensitive(assetList []string, asset string) bool {
	for _, a := range assetList {
		if strings.ToLower(asset) == strings.ToLower(a) {
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

func CheckAndPanic(e error) {
	if e != nil {
		_, file, line, _ := runtime.Caller(1) // The line that called this function
		shortFile := ShortenPoolFilePath(file, "", 0)
		log.WithField("caller", fmt.Sprintf("%s:%d", shortFile, line)).WithError(e).Fatal("An error was encountered")
	}
}

func DetailError(e error) error {
	_, file, line, _ := runtime.Caller(1) // The line that called this function
	shortFile := ShortenPoolFilePath(file, "", 0)
	return fmt.Errorf("%s:%d %s", shortFile, line, e.Error())
}

// ShortenPoolFilePath takes a long path url to prosper-pool, and shortens it:
//	"/home/billy/go/src/github.com/pegnet/pegnet/opr.go" -> "pegnet/opr.go"
//	This is nice for errors that print the file + line number
//
// 		!! Only use for error printing !!
//
func ShortenPoolFilePath(path, acc string, depth int) (trimmed string) {
	if depth > 5 || path == "." {
		// Recursive base case
		// If depth > 5 probably no pegnet dir exists
		return filepath.ToSlash(filepath.Join(path, acc))
	}
	dir, base := filepath.Split(path)
	if strings.ToLower(base) == "prosper-pool" {
		return filepath.ToSlash(filepath.Join(base, acc))
	}

	return ShortenPoolFilePath(filepath.Clean(dir), filepath.Join(base, acc), depth+1)
}

func FindIndexInStringArray(haystack []string, needle string) int {
	for i, v := range haystack {
		if v == needle {
			return i
		}
	}
	return -1
}
