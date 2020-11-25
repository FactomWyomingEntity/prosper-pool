package config

// Hex ad98d39f002d4cae9ed07a8f5689cb029a83ad3b4bd8d23c49345d4ca7ca4393
var OPRChain [32]byte = [32]byte{0xad, 0x98, 0xd3, 0x9f, 0x00, 0x2d, 0x4c,
	0xae, 0x9e, 0xd0, 0x7a, 0x8f, 0x56, 0x89, 0xcb, 0x02, 0x9a, 0x83, 0xad,
	0x3b, 0x4b, 0xd8, 0xd2, 0x3c, 0x49, 0x34, 0x5d, 0x4c, 0xa7, 0xca, 0x43,
	0x93}

// Hex 2ac925fe946543a83d4c232d788dd589177611c0dbe970172c21b42039682a8a
var TransactionChain [32]byte = [32]byte{0x2a, 0xc9, 0x25, 0xfe, 0x94, 0x65,
	0x43, 0xa8, 0x3d, 0x4c, 0x23, 0x2d, 0x78, 0x8d, 0xd5, 0x89, 0x17, 0x76,
	0x11, 0xc0, 0xdb, 0xe9, 0x70, 0x17, 0x2c, 0x21, 0xb4, 0x20, 0x39, 0x68,
	0x2a, 0x8a}

var PegnetActivation uint32 = 206421
var GradingV2Activation uint32 = 210330

// TransactionConversionActivation indicates when tx/conversions go live on mainnet.
// Target Activation Height is Oct 7, 2019 15 UTC
var TransactionConversionActivation uint32 = 213237

// Estimated to be Oct 14 2019, 15:00:00 UTC
var PEGPricingActivation uint32 = 214287

// Estimated to be  Dec 9, 2019, 17:00 UTC
var FreeFloatingPEGPriceActivation uint32 = 222270

// V4OPRUpdate indicates the activation of additional currencies and ecdsa keys.
// Estimated to be  Feb 12, 2020, 18:00 UTC
var V4OPRActivation uint32 = 231620

// V20HeightActivation indicates the activation of PegNet 2.0.
// Estimated to be  Aug 19th 2020 14:00 UTC
var V20HeightActivation uint32 = 258796

func OPRVersion(height uint32, isTestNet bool) uint8 {
	if isTestNet {
		return 5
	}
	if height < FreeFloatingPEGPriceActivation {
		return 2
	}
	if height < V4OPRActivation {
		return 3
	}
	if height < V20HeightActivation {
		return 4
	}
	return 5
}

// Compiled in
var CompiledInBuild string = "Unknown"
var CompiledInVersion string = "Unknown"
