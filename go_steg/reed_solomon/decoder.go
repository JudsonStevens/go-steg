package reed_solomon

import (
	"errors"
)

var (
	errTooManyErrors = errors.New("reed_solomon: too many errors to correct")
	errChienSearch   = errors.New("reed_solomon: error locator degree mismatch (Chien search failed)")
)

// decodeBlock decodes a Reed-Solomon codeword, correcting up to nsym/2 errors.
func decodeBlock(codeword []byte, nsym int) ([]byte, error) {
	initTables()

	syndromes := computeSyndromes(codeword, nsym)
	allZero := true
	for _, s := range syndromes {
		if s != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		out := make([]byte, len(codeword))
		copy(out, codeword)
		return out, nil
	}

	errLoc, err := berlekampMassey(syndromes, nsym)
	if err != nil {
		return nil, err
	}

	errPos, err := chienSearch(errLoc, len(codeword))
	if err != nil {
		return nil, err
	}

	corrected := make([]byte, len(codeword))
	copy(corrected, codeword)
	if err := forney(corrected, syndromes, errLoc, errPos); err != nil {
		return nil, err
	}
	return corrected, nil
}

// berlekampMassey finds the error locator polynomial from syndromes.
func berlekampMassey(syndromes []byte, nsym int) ([]byte, error) {
	// C = current connection (error locator) polynomial
	// B = previous connection polynomial
	// L = current number of errors
	C := make([]byte, nsym+1)
	B := make([]byte, nsym+1)
	C[0] = 1
	B[0] = 1
	L := 0
	m := 1 // shift counter
	b := byte(1) // previous discrepancy

	for n := 0; n < nsym; n++ {
		// Compute discrepancy d
		d := syndromes[n]
		for i := 1; i <= L; i++ {
			d ^= gfMul(C[i], syndromes[n-i])
		}

		if d == 0 {
			m++
			continue
		}

		// T = C - (d/b) * x^m * B
		T := make([]byte, nsym+1)
		copy(T, C)
		coeff := gfDiv(d, b)
		for i := 0; i+m < nsym+1; i++ {
			T[i+m] ^= gfMul(coeff, B[i])
		}

		if 2*L <= n {
			// Update B and L
			copy(B, C)
			L = n + 1 - L
			b = d
			m = 1
		} else {
			m++
		}
		copy(C, T)
	}

	// Extract the polynomial (trim trailing zeros)
	result := make([]byte, L+1)
	copy(result, C[:L+1])

	if L*2 > nsym {
		return nil, errTooManyErrors
	}

	return result, nil
}

// chienSearch finds error positions by evaluating the error locator polynomial.
// errLoc is in low-to-high order: errLoc[0] + errLoc[1]*x + errLoc[2]*x^2 + ...
func chienSearch(errLoc []byte, n int) ([]int, error) {
	numErrors := len(errLoc) - 1
	positions := make([]int, 0, numErrors)

	for i := 0; i < n; i++ {
		// Evaluate errLoc at alpha^(-i) using low-to-high accumulation
		var alphaInv byte
		if i == 0 {
			alphaInv = 1
		} else {
			alphaInv = expTable[255-i]
		}

		val := byte(0)
		xiPow := byte(1) // (alpha^(-i))^j, starting at j=0
		for j := 0; j < len(errLoc); j++ {
			val ^= gfMul(errLoc[j], xiPow)
			xiPow = gfMul(xiPow, alphaInv)
		}

		if val == 0 {
			positions = append(positions, n-1-i)
		}
	}

	if len(positions) != numErrors {
		return nil, errChienSearch
	}

	return positions, nil
}

// forney computes error magnitudes and corrects the codeword.
func forney(codeword []byte, syndromes []byte, errLoc []byte, errPos []int) error {
	nsym := len(syndromes)

	// Build syndrome polynomial S(x) = S0 + S1*x + S2*x^2 + ...
	// represented in our convention as [S_{nsym-1}, ..., S1, S0] (highest degree first)
	// But for the product we need to work coefficient-by-coefficient.

	// Compute error evaluator Omega(x) = S(x)*Lambda(x) mod x^nsym
	// where S(x) = sum(syndromes[i] * x^i) and Lambda(x) = sum(errLoc[j] * x^j)
	// Note: errLoc is stored as [Lambda_0, Lambda_1, ..., Lambda_L] (low degree first in our BM output)
	// Actually, our BM stores errLoc[0]=1, errLoc[1]=sigma_1, etc. This is low-to-high.

	// Product coefficients (low-to-high), truncated to nsym terms
	omega := make([]byte, nsym)
	for i := 0; i < nsym; i++ {
		val := byte(0)
		for j := 0; j <= i && j < len(errLoc); j++ {
			val ^= gfMul(errLoc[j], syndromes[i-j])
		}
		omega[i] = val
	}

	// Formal derivative of errLoc (low-to-high form)
	// In GF(2^m), d/dx of sum(c_i * x^i) = sum(c_i * i * x^(i-1))
	// But in characteristic 2, i is 0 for even i and 1 for odd i.
	// So Lambda'(x) = sum(c_i * x^(i-1)) for odd i only
	errLocPrime := make([]byte, len(errLoc)-1)
	for i := 1; i < len(errLoc); i++ {
		if i%2 == 1 {
			errLocPrime[i-1] = errLoc[i]
		}
	}

	// Correct each error position
	n := len(codeword)
	for _, pos := range errPos {
		if pos < 0 || pos >= n {
			return errors.New("reed_solomon: error position out of range")
		}

		// pos is array position; degree position = n-1-pos
		// X_i = alpha^(n-1-pos), X_i^{-1} = alpha^(-(n-1-pos))
		degPos := n - 1 - pos
		var xiInv byte
		if degPos == 0 {
			xiInv = 1
		} else {
			xiInv = expTable[255-degPos]
		}

		// Evaluate Omega(X_i^{-1}) using low-to-high coefficients
		omegaVal := byte(0)
		xiInvPow := byte(1)
		for j := 0; j < len(omega); j++ {
			omegaVal ^= gfMul(omega[j], xiInvPow)
			xiInvPow = gfMul(xiInvPow, xiInv)
		}

		// Evaluate Lambda'(X_i^{-1})
		derivVal := byte(0)
		xiInvPow = byte(1)
		for j := 0; j < len(errLocPrime); j++ {
			derivVal ^= gfMul(errLocPrime[j], xiInvPow)
			xiInvPow = gfMul(xiInvPow, xiInv)
		}

		if derivVal == 0 {
			return errors.New("reed_solomon: derivative is zero at error position")
		}

		// Error magnitude: e_i = X_i * Omega(X_i^{-1}) / Lambda'(X_i^{-1})
		magnitude := gfMul(expTable[degPos], gfDiv(omegaVal, derivVal))
		codeword[pos] ^= magnitude
	}

	return nil
}
