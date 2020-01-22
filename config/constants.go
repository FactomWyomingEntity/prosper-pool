package config

// Hex a642a8674f46696cc47fdb6b65f9c87b2a19c5ea8123b3d2f0c13b6f33a9d5ef
var OPRChain [32]byte = [32]byte{0xa6, 0x42, 0xa8, 0x67, 0x4f, 0x46, 0x69,
	0x6c, 0xc4, 0x7f, 0xdb, 0x6b, 0x65, 0xf9, 0xc8, 0x7b, 0x2a, 0x19, 0xc5,
	0xea, 0x81, 0x23, 0xb3, 0xd2, 0xf0, 0xc1, 0x3b, 0x6f, 0x33, 0xa9, 0xd5,
	0xef}

// Hex cffce0f409ebba4ed236d49d89c70e4bd1f1367d86402a3363366683265a242d
var TransactionChain [32]byte = [32]byte{0xcf, 0xfc, 0xe0, 0xf4, 0x09, 0xeb,
	0xba, 0x4e, 0xd2, 0x36, 0xd4, 0x9d, 0x89, 0xc7, 0x0e, 0x4b, 0xd1, 0xf1,
	0x36, 0x7d, 0x86, 0x40, 0x2a, 0x33, 0x63, 0x36, 0x66, 0x83, 0x26, 0x5a,
	0x24, 0x2d}

var PegnetActivation uint32 = 206421
var GradingV2Activation uint32 = 210330

// TransactionConversionActivation indicates when tx/conversions go live on mainnet.
// Target Activation Height is Oct 7, 2019 15 UTC
var TransactionConversionActivation uint32 = 213237

// Estimated to be Oct 14 2019, 15:00:00 UTC
var PEGPricingActivation uint32 = 214287

// Estimated to be  Dec 9, 2019, 17:00 UTC
var FreeFloatingPEGPriceActivation uint32 = 222270

var V4OPRActivation uint32 = 999999

func OPRVersion(height uint32) uint8 {
	if height < FreeFloatingPEGPriceActivation {
		return 2
	}
	if height < V4OPRActivation {
		return 3
	}
	return 4
}

// Compiled in
var CompiledInBuild string = "Unknown"
var CompiledInVersion string = "Unknown"
