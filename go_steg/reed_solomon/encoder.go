package reed_solomon

// generatorPoly builds g(x) = product of (x - alpha^i) for i=0..nsym-1
func generatorPoly(nsym int) []byte {
	initTables()
	g := []byte{1}
	for i := 0; i < nsym; i++ {
		g = polyMul(g, []byte{1, expTable[i]})
	}
	return g
}

func polyMul(a, b []byte) []byte {
	result := make([]byte, len(a)+len(b)-1)
	for i, av := range a {
		for j, bv := range b {
			result[i+j] ^= gfMul(av, bv)
		}
	}
	return result
}

// encodeBlock computes parity bytes via polynomial long division
func encodeBlock(data []byte, nsym int) []byte {
	gen := generatorPoly(nsym)
	padded := make([]byte, len(data)+nsym)
	copy(padded, data)
	for i := 0; i < len(data); i++ {
		coef := padded[i]
		if coef != 0 {
			for j := 1; j < len(gen); j++ {
				padded[i+j] ^= gfMul(gen[j], coef)
			}
		}
	}
	return padded[len(data):]
}

// computeSyndromes evaluates codeword at alpha^0..alpha^(nsym-1)
func computeSyndromes(codeword []byte, nsym int) []byte {
	initTables()
	syndromes := make([]byte, nsym)
	for i := 0; i < nsym; i++ {
		val := byte(0)
		for _, c := range codeword {
			val = gfMul(val, expTable[i]) ^ c
		}
		syndromes[i] = val
	}
	return syndromes
}
