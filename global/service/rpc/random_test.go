package rpc

// func TestRandomServer(t *testing.T) {
// 	t.Run("QueryRandomNumber", func(t *testing.T) {
// 		srv := &RandomServer{
// 			client: crypto.NewBlsClient(),
// 		}
// 		err := srv.init()
// 		assert.Nil(t, err, "init error")
// 		req := &pb.QueryRandomNumberRequest{
// 			Height: 1,
// 		}

// 		// get random number by height
// 		resp, err := srv.QueryRandomNumber(context.Background(), req)
// 		if err != nil {
// 			t.Errorf("QueryRandomNumber failed: %v", err)
// 		}

// 		t.Log(resp.RandomNumber)
// 		t.Log(resp.Proof)

// 		// verify proof locally
// 		sign, err := hex.DecodeString(resp.RandomNumber)
// 		assert.Nil(t, err, "decode random number error")
// 		blsProof := bls.Proof{
// 			Message:          string(resp.Proof.Message),
// 			PPrime:           bls.PublicKey(resp.Proof.PPrime),
// 			PartIndexes:      resp.Proof.Indexes,
// 			PartPublicKeySum: bls.PublicKey(resp.Proof.PartPublicKeySum),
// 		}
// 		result := srv.client.VerifySignatureByProof(sign, blsProof)
// 		assert.True(t, result, "proof failed")
// 	})
// }
