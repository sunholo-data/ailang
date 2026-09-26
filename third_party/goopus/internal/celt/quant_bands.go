// Transliteration of celt/quant_bands.c (libopus v1.6.1, commit 3da9f7a6) for
// the frozen FIXED_POINT + DISABLE_FLOAT_API build: band energy quantization,
// both directions. Decode side (phase 2): unquantCoarseEnergy decodes the coarse
// (Laplace-coded) log energies with the inter/intra prediction filter over
// oldEBands; unquantFineEnergy / unquantEnergyFinalise add the refinement bits.
// Encode side (phase 4): amp2Log2 turns band amplitudes into log energies;
// quantCoarseEnergy (with the two-pass intra/inter RDO in quantCoarseEnergyImpl),
// quantFineEnergy and quantEnergyFinalise are the mirror encoders. The file has
// no ENABLE_QEXT branch in this tag, so there is no QEXT boundary to gate here
// (the prev_quant fine-energy parameter stays nil on the frozen path).
//
// Cross-frame contract (docs/hard-parts.md 5): oldEBands is both the prediction
// base (input) and the reconstruction (output), and delayedIntra carries the
// intra/inter RDO bias across frames. The caller carries both; the coarse coders
// clamp and overwrite oldEBands in place. This is why the differential tests run
// multi-frame sequences with a per-frame state comparison, never single frames.

package celt

import (
	"github.com/sunholo-data/ailang/third_party/goopus/internal/fixedmath"
	"github.com/sunholo-data/ailang/third_party/goopus/internal/rangecoding"
)

// dbShift is celt/arch.h DB_SHIFT: the fractional bits of a celt_glog log-energy
// value (Q24) in the FIXED_POINT build (arch.h:219).
const dbShift = 24

// maxFineBits is celt/rate.h MAX_FINE_BITS: the cap on fine-energy bits per band.
const maxFineBits = 8

// Fixed-point log-energy constants, GCONST(x) = QCONST32(x, DB_SHIFT). Computed
// with the same compile-time float-to-fixed rounding as the C macro
// (celt/fixed_generic.h:98-101), evaluated once at package init.
// gconst2 (GCONST(2.f)) is declared in celt_decoder.go and reused here.
var (
	gconst3  = fixedmath.QCONST32(3, dbShift)   // GCONST(3.f)
	gconst9  = fixedmath.QCONST32(9, dbShift)   // GCONST(9.f)
	gconst14 = fixedmath.QCONST32(14, dbShift)  // GCONST(14.f)
	gconst28 = fixedmath.QCONST32(28, dbShift)  // GCONST(28.f)
	ghalf    = fixedmath.QCONST32(0.5, dbShift) // GCONST(.5f)
)

// Prediction coefficients 0.9, 0.8, 0.65, 0.5 in Q15, indexed by LM
// (celt/quant_bands.c:61-65, FIXED_POINT). The decoder uses the same constants
// as the encoder: coef predicts from the previous frame's energy (inter),
// betaCoef/betaIntra drive the intra-frame leakage into prev[].
var (
	predCoef = [4]int16{29440, 26112, 21248, 16384}
	betaCoef = [4]int16{30147, 22282, 12124, 6554}
)

// betaIntra is beta_intra, the intra-frame beta coefficient in Q15
// (celt/quant_bands.c:65).
const betaIntra int16 = 4915

// eProbModel transcribes e_prob_model[4][2][42] (celt/quant_bands.c:77).
// Laplace-like probability models for the coarse energy, one pair per frame
// size (LM), prediction type (0=inter, 1=intra) and band: the first of each
// pair is the probability of 0, the second the decay rate, both in Q8.
var eProbModel = [4][2][42]uint8{
	// 120-sample frames (LM=0).
	{
		{ // Inter.
			72, 127, 65, 129, 66, 128, 65, 128, 64, 128, 62, 128, 64, 128,
			64, 128, 92, 78, 92, 79, 92, 78, 90, 79, 116, 41, 115, 40,
			114, 40, 132, 26, 132, 26, 145, 17, 161, 12, 176, 10, 177, 11,
		},
		{ // Intra.
			24, 179, 48, 138, 54, 135, 54, 132, 53, 134, 56, 133, 55, 132,
			55, 132, 61, 114, 70, 96, 74, 88, 75, 88, 87, 74, 89, 66,
			91, 67, 100, 59, 108, 50, 120, 40, 122, 37, 97, 43, 78, 50,
		},
	},
	// 240-sample frames (LM=1).
	{
		{ // Inter.
			83, 78, 84, 81, 88, 75, 86, 74, 87, 71, 90, 73, 93, 74,
			93, 74, 109, 40, 114, 36, 117, 34, 117, 34, 143, 17, 145, 18,
			146, 19, 162, 12, 165, 10, 178, 7, 189, 6, 190, 8, 177, 9,
		},
		{ // Intra.
			23, 178, 54, 115, 63, 102, 66, 98, 69, 99, 74, 89, 71, 91,
			73, 91, 78, 89, 86, 80, 92, 66, 93, 64, 102, 59, 103, 60,
			104, 60, 117, 52, 123, 44, 138, 35, 133, 31, 97, 38, 77, 45,
		},
	},
	// 480-sample frames (LM=2).
	{
		{ // Inter.
			61, 90, 93, 60, 105, 42, 107, 41, 110, 45, 116, 38, 113, 38,
			112, 38, 124, 26, 132, 27, 136, 19, 140, 20, 155, 14, 159, 16,
			158, 18, 170, 13, 177, 10, 187, 8, 192, 6, 175, 9, 159, 10,
		},
		{ // Intra.
			21, 178, 59, 110, 71, 86, 75, 85, 84, 83, 91, 66, 88, 73,
			87, 72, 92, 75, 98, 72, 105, 58, 107, 54, 115, 52, 114, 55,
			112, 56, 129, 51, 132, 40, 150, 33, 140, 29, 98, 35, 77, 42,
		},
	},
	// 960-sample frames (LM=3).
	{
		{ // Inter.
			42, 121, 96, 66, 108, 43, 111, 40, 117, 44, 123, 32, 120, 36,
			119, 33, 127, 33, 134, 34, 139, 21, 147, 23, 152, 20, 158, 25,
			154, 26, 166, 21, 173, 16, 184, 13, 184, 10, 150, 13, 139, 15,
		},
		{ // Intra.
			22, 178, 63, 114, 74, 82, 84, 83, 92, 82, 103, 62, 96, 72,
			96, 67, 101, 73, 107, 72, 113, 55, 118, 52, 125, 52, 118, 52,
			117, 55, 135, 49, 137, 39, 157, 32, 145, 29, 97, 33, 77, 40,
		},
	},
}

// smallEnergyICDF is small_energy_icdf (celt/quant_bands.c:140): the 3-entry
// inverse CDF used to code a coarse-energy delta when only 2-14 bits remain.
var smallEnergyICDF = [3]byte{2, 1, 0}

// unquantCoarseEnergy ports unquant_coarse_energy (celt/quant_bands.c:431). It
// decodes the coarse log energies into oldEBands using the inter/intra
// prediction filter, clamping each reconstructed value to [-28, 28] in Q24.
// prev[] is opus_val64 (int64) as in C; the intermediate sum is formed in int64
// and narrowed to the int32 celt_glog exactly like the C assignment.
func unquantCoarseEnergy(m *celtMode, start, end int, oldEBands []int32, intra int, dec *rangecoding.Decoder, C, LM int) {
	probModel := eProbModel[LM][intra]
	var prev [2]int64 // opus_val64 prev[2]
	var coef, beta int16

	if intra != 0 {
		coef = 0
		beta = betaIntra
	} else {
		beta = betaCoef[LM]
		coef = predCoef[LM]
	}

	budget := int32(dec.Storage() * 8)

	// Decode at a fixed coarse resolution.
	for i := start; i < end; i++ {
		c := 0
		for {
			var qi int
			tell := int32(dec.Tell())
			switch {
			case budget-tell >= 15:
				pi := 2 * fixedmath.IMIN(i, 20)
				qi = ecLaplaceDecode(dec, uint32(probModel[pi])<<7, int(probModel[pi+1])<<6)
			case budget-tell >= 2:
				qi = dec.DecIcdf(smallEnergyICDF[:], 2)
				qi = (qi >> 1) ^ -(qi & 1)
			case budget-tell >= 1:
				qi = -dec.DecBitLogp(1)
			default:
				qi = -1
			}
			q := fixedmath.SHL32(int32(qi), dbShift)

			idx := i + c*m.nbEBands
			oldEBands[idx] = fixedmath.MAX32(-gconst9, oldEBands[idx])
			tmp := int32(int64(fixedmath.MULT16_32_Q15(coef, oldEBands[idx])) + prev[c] + int64(q))
			tmp = fixedmath.MIN32(gconst28, fixedmath.MAX32(-gconst28, tmp))
			oldEBands[idx] = tmp
			prev[c] = prev[c] + int64(q) - int64(fixedmath.MULT16_32_Q15(beta, q))

			c++
			if c >= C {
				break
			}
		}
	}
}

// unquantFineEnergy ports unquant_fine_energy (celt/quant_bands.c:496). It reads
// extraQuant[i] raw bits per band (per channel) and adds the fine-resolution
// offset into oldEBands. prevQuant is the QEXT "previous fine bits" array (NULL
// on the main path); it is honored for faithfulness though the non-QEXT decoder
// always passes nil.
func unquantFineEnergy(m *celtMode, start, end int, oldEBands []int32, prevQuant, extraQuant []int, dec *rangecoding.Decoder, C int) {
	// Decode finer resolution.
	for i := start; i < end; i++ {
		extra := int16(extraQuant[i])
		if extraQuant[i] <= 0 {
			continue
		}
		if int32(dec.Tell())+int32(C*extraQuant[i]) > int32(dec.Storage()*8) {
			continue
		}
		var prev int16
		if prevQuant != nil {
			prev = int16(prevQuant[i])
		}
		c := 0
		for {
			q2 := int32(dec.DecBits(uint32(extra)))
			// Has to be without rounding.
			offset := fixedmath.SUB32(fixedmath.VSHR32(2*q2+1, int(extra)-dbShift+1), ghalf)
			offset = fixedmath.SHR32(offset, int(prev))
			oldEBands[i+c*m.nbEBands] += offset
			c++
			if c >= C {
				break
			}
		}
	}
}

// unquantEnergyFinalise ports unquant_energy_finalise (celt/quant_bands.c:525).
// It uses up the remaining bits, one per (band, channel), in two priority
// passes, adding a +/-0.5-LSB offset into oldEBands. oldEBands may be nil (the
// QEXT path), in which case the bits are consumed but discarded.
func unquantEnergyFinalise(m *celtMode, start, end int, oldEBands []int32, fineQuant, finePriority []int, bitsLeft int, dec *rangecoding.Decoder, C int) {
	// Use up the remaining bits.
	for prio := 0; prio < 2; prio++ {
		for i := start; i < end && bitsLeft >= C; i++ {
			if fineQuant[i] >= maxFineBits || finePriority[i] != prio {
				continue
			}
			c := 0
			for {
				q2 := int32(dec.DecBits(1))
				offset := fixedmath.SHR32(fixedmath.SHL32(q2, dbShift)-ghalf, fineQuant[i]+1)
				if oldEBands != nil {
					oldEBands[i+c*m.nbEBands] += offset
				}
				bitsLeft--
				c++
				if c >= C {
					break
				}
			}
		}
	}
}

// lossDistortion ports the static loss_distortion (celt/quant_bands.c:142): the
// squared log-energy error between the true band energies and the prediction
// base, used to bias the intra/inter RDO. length is m->nbEBands. The per-band
// difference is truncated to opus_val16 for the MAC16_16, exactly as the C does.
func lossDistortion(eBands, oldEBands []int32, start, end, length, C int) int32 {
	var dist int32
	c := 0
	for {
		for i := start; i < end; i++ {
			d := fixedmath.PSHR32(fixedmath.SUB32(eBands[i+c*length], oldEBands[i+c*length]), dbShift-7)
			dist = fixedmath.MAC16_16(dist, int16(d), int16(d))
		}
		c++
		if c >= C {
			break
		}
	}
	return fixedmath.MIN32(200, fixedmath.SHR32(dist, 14))
}

// quantCoarseEnergyImpl ports quant_coarse_energy_impl (celt/quant_bands.c:156):
// a single coarse-energy pass (intra or inter) over the bands. It writes the
// intra flag, Laplace-codes each per-band quantized delta qi (with the small
// budget-limited fallbacks), accumulates error[] and the reconstructed
// oldEBands, and returns the "badness" (sum of |qi0-qi| clamping) that the RDO
// decision compares. prev[] is opus_val32 here (not opus_val64 as in the
// decoder). This is Layer C coded-output: transliterated verbatim.
func quantCoarseEnergyImpl(m *celtMode, start, end int, eBands, oldEBands []int32,
	budget, tell int32, probModel []uint8, error []int32, enc *rangecoding.Encoder,
	C, LM, intra int, maxDecay int32, lfe int) int {
	badness := 0
	var prev [2]int32
	var coef, beta int16

	if tell+3 <= budget {
		enc.EncBitLogp(intra, 3)
	}
	if intra != 0 {
		coef = 0
		beta = betaIntra
	} else {
		beta = betaCoef[LM]
		coef = predCoef[LM]
	}

	// Encode at a fixed coarse resolution.
	for i := start; i < end; i++ {
		c := 0
		for {
			var qi, qi0 int
			var q, tmp int32
			var oldE, decayBound int32
			x := eBands[i+c*m.nbEBands]
			oldE = fixedmath.MAX32(-gconst9, oldEBands[i+c*m.nbEBands])
			f := x - fixedmath.MULT16_32_Q15(coef, oldE) - prev[c]
			// Rounding to nearest integer here is really important!
			qi = int(fixedmath.SHR32(f+ghalf, dbShift))
			decayBound = fixedmath.MAX32(-gconst28, fixedmath.SUB32(oldEBands[i+c*m.nbEBands], maxDecay))
			// Prevent the energy from going down too quickly (e.g. for bands
			// that have just one bin).
			if qi < 0 && x < decayBound {
				qi += int(fixedmath.SHR32(fixedmath.SUB32(decayBound, x), dbShift))
				if qi > 0 {
					qi = 0
				}
			}
			qi0 = qi
			// If we don't have enough bits to encode all the energy, just
			// assume something safe.
			tell = int32(enc.Tell())
			bitsLeft := int(budget) - int(tell) - 3*C*(end-i)
			if i != start && bitsLeft < 30 {
				if bitsLeft < 24 {
					qi = fixedmath.IMIN(1, qi)
				}
				if bitsLeft < 16 {
					qi = fixedmath.IMAX(-1, qi)
				}
			}
			if lfe != 0 && i >= 2 {
				qi = fixedmath.IMIN(qi, 0)
			}
			if budget-tell >= 15 {
				pi := 2 * fixedmath.IMIN(i, 20)
				ecLaplaceEncode(enc, &qi, uint32(probModel[pi])<<7, int(probModel[pi+1])<<6)
			} else if budget-tell >= 2 {
				qi = fixedmath.IMAX(-1, fixedmath.IMIN(qi, 1))
				sign := 0
				if qi < 0 {
					sign = -1
				}
				enc.EncIcdf((2*qi)^sign, smallEnergyICDF[:], 2)
			} else if budget-tell >= 1 {
				qi = fixedmath.IMIN(0, qi)
				enc.EncBitLogp(-qi, 1)
			} else {
				qi = -1
			}
			error[i+c*m.nbEBands] = f - fixedmath.SHL32(int32(qi), dbShift)
			d := qi0 - qi
			if d < 0 {
				d = -d
			}
			badness += d
			q = fixedmath.SHL32(fixedmath.EXTEND32(int16(qi)), dbShift)

			tmp = fixedmath.MULT16_32_Q15(coef, oldE) + prev[c] + q
			tmp = fixedmath.MAX32(-gconst28, tmp)
			oldEBands[i+c*m.nbEBands] = tmp
			prev[c] = prev[c] + q - fixedmath.MULT16_32_Q15(beta, q)

			c++
			if c >= C {
				break
			}
		}
	}
	if lfe != 0 {
		return 0
	}
	return badness
}

// quantCoarseEnergy ports quant_coarse_energy (celt/quant_bands.c:260): the
// two-pass intra/inter RDO wrapper. It snapshots the range coder, runs the intra
// pass into a scratch oldEBands_intra, restores, runs the inter pass into
// oldEBands, and if intra wins (by badness, then by a tell_frac + intra_bias
// tie-break) it restores the intra encoder state AND splices the saved intra
// bytes back over the shared buffer window. delayedIntra carries the RDO bias
// across frames. This is the first range-coder snapshot/splice site
// (docs/hard-parts.md 1); see the comment on rangecoding.Encoder.
//
// It returns the intra decision it settled on (0 = inter, 1 = intra). C returns
// void; the value is exposed only so the celt_encode_with_ec differential test
// can prove non-vacuously that the intra branch fired (it is otherwise buried in
// the bitstream), and no caller acts on it.
func quantCoarseEnergy(m *celtMode, start, end, effEnd int, eBands, oldEBands []int32,
	budget uint32, error []int32, enc *rangecoding.Encoder, C, LM, nbAvailableBytes,
	forceIntra int, delayedIntra *int32, twoPass, lossRate, lfe int, sc *scratch) int {
	var intra int
	var maxDecay int32

	if forceIntra != 0 || (twoPass == 0 && *delayedIntra > int32(2*C*(end-start)) && nbAvailableBytes > (end-start)*C) {
		intra = 1
	}
	intraBias := int32(budget * uint32(*delayedIntra) * uint32(lossRate) / uint32(C*512))
	newDistortion := lossDistortion(eBands, oldEBands, start, effEnd, m.nbEBands, C)

	tell := int32(enc.Tell())
	if tell+3 > int32(budget) {
		twoPass = 0
		intra = 0
	}

	maxDecay = gconst16
	if end-start > 10 {
		maxDecay = fixedmath.SHL32(fixedmath.MIN32(fixedmath.SHR32(maxDecay, dbShift-3), fixedmath.EXTEND32(int16(nbAvailableBytes))), dbShift-3)
	}
	if lfe != 0 {
		maxDecay = gconst3
	}
	encStartState := *enc

	oldEBandsIntra := alloc(&sc.oldEBandsIntra, C*m.nbEBands) // VARDECL(celt_glog, oldEBands_intra)
	errorIntra := alloc(&sc.errorIntra, C*m.nbEBands)         // VARDECL(celt_glog, error_intra)
	copy(oldEBandsIntra, oldEBands[:C*m.nbEBands])

	badness1 := 0
	if twoPass != 0 || intra != 0 {
		badness1 = quantCoarseEnergyImpl(m, start, end, eBands, oldEBandsIntra, int32(budget),
			tell, eProbModel[LM][1][:], errorIntra, enc, C, LM, 1, maxDecay, lfe)
	}

	if intra == 0 {
		tellIntra := int32(enc.TellFrac())

		encIntraState := *enc

		nstartBytes := encStartState.RangeBytes()
		nintraBytes := encIntraState.RangeBytes()
		// intra_buf = ec_get_buffer(&enc_intra_state) + nstart_bytes: the head
		// window that the intra pass wrote. A nil slice is C's ALLOC_NONE.
		var intraBits []byte
		if nintraBytes > nstartBytes {
			intraBits = alloc(&sc.intraBits, int(nintraBytes-nstartBytes)) // VARDECL(unsigned char, intra_bits)
			// Copy bits from intra bit-stream.
			copy(intraBits, enc.Buffer()[nstartBytes:nintraBytes])
		}

		*enc = encStartState

		badness2 := quantCoarseEnergyImpl(m, start, end, eBands, oldEBands, int32(budget),
			tell, eProbModel[LM][intra][:], error, enc, C, LM, 0, maxDecay, lfe)

		if twoPass != 0 && (badness1 < badness2 || (badness1 == badness2 && int32(enc.TellFrac())+intraBias > tellIntra)) {
			*enc = encIntraState
			// Copy intra bits to bit-stream.
			if nintraBytes > nstartBytes {
				copy(enc.Buffer()[nstartBytes:nintraBytes], intraBits)
			}
			copy(oldEBands[:C*m.nbEBands], oldEBandsIntra)
			copy(error[:C*m.nbEBands], errorIntra)
			intra = 1
		}
	} else {
		copy(oldEBands[:C*m.nbEBands], oldEBandsIntra)
		copy(error[:C*m.nbEBands], errorIntra)
	}

	if intra != 0 {
		*delayedIntra = newDistortion
	} else {
		*delayedIntra = fixedmath.ADD32(fixedmath.MULT16_32_Q15(int16(fixedmath.MULT16_16_Q15(predCoef[LM], predCoef[LM])), *delayedIntra),
			newDistortion)
	}
	return intra
}

// quantFineEnergy ports quant_fine_energy (celt/quant_bands.c:360): the
// fine-resolution refinement. Per band (skipping bands with no fine bits or that
// would overrun the budget) it codes extra_quant[i] raw bits per channel and
// folds the offset into oldEBands, subtracting it from error. prevQuant is the
// QEXT "previous fine bits" array (nil on the frozen path).
func quantFineEnergy(m *celtMode, start, end int, oldEBands, error []int32, prevQuant, extraQuant []int, enc *rangecoding.Encoder, C int) {
	// Encode finer resolution.
	for i := start; i < end; i++ {
		extra := int16(1 << extraQuant[i])
		if extraQuant[i] <= 0 {
			continue
		}
		if int32(enc.Tell())+int32(C*extraQuant[i]) > int32(enc.Storage()*8) {
			continue
		}
		var prev int16
		if prevQuant != nil {
			prev = int16(prevQuant[i])
		}
		c := 0
		for {
			// Has to be without rounding.
			q2 := fixedmath.VSHR32(fixedmath.ADD32(error[i+c*m.nbEBands], fixedmath.SHR32(ghalf, int(prev))), dbShift-extraQuant[i]-int(prev))
			if q2 > int32(extra)-1 {
				q2 = int32(extra) - 1
			}
			if q2 < 0 {
				q2 = 0
			}
			enc.EncBits(uint32(q2), uint32(extraQuant[i]))
			offset := fixedmath.SUB32(fixedmath.VSHR32(2*q2+1, extraQuant[i]-dbShift+1), ghalf)
			offset = fixedmath.SHR32(offset, int(prev))
			oldEBands[i+c*m.nbEBands] += offset
			error[i+c*m.nbEBands] -= offset
			c++
			if c >= C {
				break
			}
		}
	}
}

// quantEnergyFinalise ports quant_energy_finalise (celt/quant_bands.c:401): it
// uses up any leftover whole bits, one per (band, channel) in two priority
// passes, coding the sign of the residual error and nudging oldEBands by a
// half-LSB. It mirrors unquant_energy_finalise on the decode side.
func quantEnergyFinalise(m *celtMode, start, end int, oldEBands, error []int32, fineQuant, finePriority []int, bitsLeft int, enc *rangecoding.Encoder, C int) {
	// Use up the remaining bits.
	for prio := 0; prio < 2; prio++ {
		for i := start; i < end && bitsLeft >= C; i++ {
			if fineQuant[i] >= maxFineBits || finePriority[i] != prio {
				continue
			}
			c := 0
			for {
				var q2 int32
				if error[i+c*m.nbEBands] < 0 {
					q2 = 0
				} else {
					q2 = 1
				}
				enc.EncBits(uint32(q2), 1)
				offset := fixedmath.SHR32(fixedmath.SHL32(q2, dbShift)-ghalf, fineQuant[i]+1)
				if oldEBands != nil {
					oldEBands[i+c*m.nbEBands] += offset
				}
				error[i+c*m.nbEBands] -= offset
				bitsLeft--
				c++
				if c >= C {
					break
				}
			}
		}
	}
}

// amp2Log2 ports amp2Log2 (celt/quant_bands.c:553): convert band amplitudes
// (celt_ener) to log energies (celt_glog) minus the per-band mean, using the
// non-QEXT celt_log2_db macro form (SHL32(EXTEND32(celt_log2(x)), DB_SHIFT-10)).
// The FIXED_POINT build adds GCONST(2.f) to compensate for bandE being Q12 while
// celt_log2 takes a Q14 input. Bands from effEnd..end are set to -GCONST(14.f).
func amp2Log2(m *celtMode, effEnd, end int, bandE, bandLogE []int32, C int) {
	c := 0
	for {
		for i := 0; i < effEnd; i++ {
			// celt_log2_db(x) = SHL32(EXTEND32(celt_log2(x)), DB_SHIFT-10).
			bandLogE[i+c*m.nbEBands] = fixedmath.SHL32(fixedmath.EXTEND32(fixedmath.Celt_log2(bandE[i+c*m.nbEBands])), dbShift-10) -
				fixedmath.SHL32(int32(eMeans[i]), dbShift-4)
			// Compensate for bandE[] being Q12 but celt_log2() taking a Q14 input.
			bandLogE[i+c*m.nbEBands] += gconst2
		}
		for i := effEnd; i < end; i++ {
			bandLogE[c*m.nbEBands+i] = -gconst14
		}
		c++
		if c >= C {
			break
		}
	}
}

// UnquantCoarseEnergy is the exported test seam over unquantCoarseEnergy bound
// to the frozen mode48000_960 (mirroring InverseMDCT's adapter shape). It lets
// the refc cgo differential harness (internal/reftest/oracle) drive the pure-Go
// coarse-energy decode against libopus; it is not part of the decoder API. C is
// the channel count (1 or 2); intra is the already-decoded intra-energy flag.
func UnquantCoarseEnergy(start, end int, oldEBands []int32, intra int, dec *rangecoding.Decoder, C, LM int) {
	unquantCoarseEnergy(&mode48000_960, start, end, oldEBands, intra, dec, C, LM)
}

// UnquantFineEnergy is the exported test seam over unquantFineEnergy, bound to
// mode48000_960 with prevQuant=nil (the non-QEXT decoder path,
// celt/celt_decoder.c:1455). fineQuant is the per-band fine bit count.
func UnquantFineEnergy(start, end int, oldEBands []int32, fineQuant []int, dec *rangecoding.Decoder, C int) {
	unquantFineEnergy(&mode48000_960, start, end, oldEBands, nil, fineQuant, dec, C)
}

// UnquantEnergyFinalise is the exported test seam over unquantEnergyFinalise
// bound to mode48000_960 (celt/celt_decoder.c:1520). bitsLeft is len*8 minus the
// bits used so far, exactly as the decoder pipeline computes it.
func UnquantEnergyFinalise(start, end int, oldEBands []int32, fineQuant, finePriority []int, bitsLeft int, dec *rangecoding.Decoder, C int) {
	unquantEnergyFinalise(&mode48000_960, start, end, oldEBands, fineQuant, finePriority, bitsLeft, dec, C)
}

// EcLaplaceEncode is the exported test seam over ecLaplaceEncode, letting the
// refc harness drive the pure-Go Laplace encoder against ec_laplace_encode.
// value is modified in place when the value lands in the PDF tail.
func EcLaplaceEncode(enc *rangecoding.Encoder, value *int, fs uint32, decay int) {
	ecLaplaceEncode(enc, value, fs, decay)
}

// QuantCoarseEnergy is the exported test seam over quantCoarseEnergy bound to
// mode48000_960 (celt/celt_encoder.c:2295). delayedIntra carries the RDO bias
// across frames; the caller holds oldEBands and error across the sequence.
func QuantCoarseEnergy(start, end, effEnd int, eBands, oldEBands []int32, budget uint32,
	error []int32, enc *rangecoding.Encoder, C, LM, nbAvailableBytes, forceIntra int,
	delayedIntra *int32, twoPass, lossRate, lfe int) {
	var sc scratch
	_ = quantCoarseEnergy(&mode48000_960, start, end, effEnd, eBands, oldEBands, budget, error,
		enc, C, LM, nbAvailableBytes, forceIntra, delayedIntra, twoPass, lossRate, lfe, &sc)
}

// QuantFineEnergy is the exported test seam over quantFineEnergy bound to
// mode48000_960 with prevQuant=nil (the non-QEXT encoder path,
// celt/celt_encoder.c:2634). extraQuant is the per-band fine bit count.
func QuantFineEnergy(start, end int, oldEBands, error []int32, extraQuant []int, enc *rangecoding.Encoder, C int) {
	quantFineEnergy(&mode48000_960, start, end, oldEBands, error, nil, extraQuant, enc, C)
}

// QuantEnergyFinalise is the exported test seam over quantEnergyFinalise bound to
// mode48000_960 (celt/celt_encoder.c:2706). bitsLeft is nbCompressedBytes*8 minus
// the bits used so far, exactly as the encoder pipeline computes it.
func QuantEnergyFinalise(start, end int, oldEBands, error []int32, fineQuant, finePriority []int, bitsLeft int, enc *rangecoding.Encoder, C int) {
	quantEnergyFinalise(&mode48000_960, start, end, oldEBands, error, fineQuant, finePriority, bitsLeft, enc, C)
}

// Amp2Log2 is the exported test seam over amp2Log2 bound to mode48000_960
// (celt/celt_encoder.c:2106). bandE is the amplitude array from
// ComputeBandEnergies; bandLogE receives the mean-removed log energies.
func Amp2Log2(effEnd, end int, bandE, bandLogE []int32, C int) {
	amp2Log2(&mode48000_960, effEnd, end, bandE, bandLogE, C)
}
