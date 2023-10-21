package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVerifySigBoring(t *testing.T) {
	const (
		msg  = "aea3eb2f5a6a2efe002d3c88da52ba5a8702c9722ae2d67d1260e4318f5ccd6c"
		sig  = "0x93433430e249145433931dd4fda65090fcb250489e107d460b8adcef4a3c05f863c860d63c3f4ff92dcda4cce9755e4771e9dc6b91dafd5c900c7a5b99c169d71b"
		from = "0xb938F65DfE303EdF96A511F1e7E3190f69036860"
	)

	res, err := verifySig(from, msg, sig)
	assert.Nil(t, err)
	assert.True(t, res)
}

func TestVerifySig(t *testing.T) {
	const (
		msg  = "Join the meme army!!"
		sig  = "0xf689927d8f85c4f11b5b7ff62cfe9c8631f1373d3fe74a168031d23a7b7faf6903cfaed61f9fd153c506ffe3e1fd912fb42271859d6307239d36272ed00947da1b"
		from = "0xb938f65dfe303edf96a511f1e7e3190f69036860"
	)

	res, err := verifySig(from, msg, sig)
	assert.Nil(t, err)
	assert.True(t, res)
}
