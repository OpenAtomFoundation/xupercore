package reader

import (
    "encoding/hex"
    "encoding/json"
    "fmt"
    cons_base "github.com/xuperchain/xuperchain/core/consensus/base"
    "github.com/xuperchain/xuperchain/core/consensus/tdpos"
    "github.com/xuperchain/xuperchain/core/pb"
    "github.com/xuperchain/xupercore/kernel/consensus/base"
    "github.com/xuperchain/xupercore/kernel/engines/xuperos/def"
    "github.com/xuperchain/xupercore/lib/logs"
)

type Consensus interface {
    GetConsType() string
    GetConsStatus() base.ConsensusStatus
    GetCheckResults(term int64) ([]string, error)

    GetDposVotedRecords(addr string) ([]*pb.VotedRecord, error)
    GetDposVoteRecords(addr string) ([]*pb.VoteRecord, error)
    GetDposNominatedRecords(addr string) (string, error)
    GetDposNominateRecords(addr string) ([]*pb.DposNominateInfo, error)
    GetDposCandidates() ([]string, error)
}

type consensusReader struct {
    ctx       *def.ChainCtx
    log       logs.Logger
    chain     def.Chain
    consensus def.XConsensus
}

func NewConsensusReader(chain def.Chain) Consensus {
    reader := &consensusReader{
        chain: chain,
        ctx:   chain.Context(),
        log:   chain.Context().XLog,
        consensus: chain.Context().Consensus,
    }

    return reader
}

func (t *consensusReader) GetConsType() string {
    return t.consensus.Status().GetConsensusName()
}

func (t *consensusReader) GetConsStatus() base.ConsensusStatus {
    return t.consensus.Status()
}

func (t *consensusReader) GetCheckResults(term int64) ([]string, error) {
    res := []string{}
    proposers := []*cons_base.CandidateInfo{}
    version := t.con.Version(t.Ledger.GetMeta().TrunkHeight + 1)
    key := tdpos.GenTermCheckKey(version, term)
    val, err := t.Utxovm.GetFromTable(nil, []byte(key))
    if err != nil || val == nil {
        return nil, err
    }
    err = json.Unmarshal(val, &proposers)
    if err != nil {
        return nil, err
    }
    for _, proposer := range proposers {
        res = append(res, proposer.Address)
    }
    return res, nil
}
func (t *consensusReader) GetDposVotedRecords(addr string) ([]*pb.VotedRecord, error) {
    votedRecords := []*pb.VotedRecord{}
    it := t.Utxovm.ScanWithPrefix([]byte(tdpos.GenCandidateVotePrefix(addr)))
    defer it.Release()
    for it.Next() {
        key := string(it.Key())
        voter, txid, err := tdpos.ParseCandidateVoteKey(key)
        votedRecord := &pb.VotedRecord{
            Voter: voter,
            Txid:  txid,
        }
        if err != nil {
            return nil, err
        }
        votedRecords = append(votedRecords, votedRecord)
    }
    return votedRecords, nil
}
func (t *consensusReader) GetDposVoteRecords(addr string) ([]*pb.VoteRecord, error) {
    voteRecords := []*pb.VoteRecord{}
    it := t.Utxovm.ScanWithPrefix([]byte(tdpos.GenVoteCandidatePrefix(addr)))
    defer it.Release()
    for it.Next() {
        key := string(it.Key())
        candidate, txid, err := tdpos.ParseVoteCandidateKey(key)
        voteRecord := &pb.VoteRecord{
            Candidate: candidate,
            Txid:      txid,
        }
        if err != nil {
            return nil, err
        }
        voteRecords = append(voteRecords, voteRecord)
    }
    return voteRecords, nil
}
func (t *consensusReader) GetDposNominatedRecords(addr string) (string, error) {
    key := tdpos.GenCandidateNominateKey(addr)
    val, err := t.Utxovm.GetFromTable(nil, []byte(key))
    if err != nil {
        return "", err
    }
    return hex.EncodeToString(val), err
}
func (t *consensusReader) GetDposNominateRecords(addr string) ([]*pb.DposNominateInfo, error) {
    nominateRecords := []*pb.DposNominateInfo{}
    it := t.Utxovm.ScanWithPrefix([]byte(tdpos.GenNominateRecordsPrefix(addr)))
    defer it.Release()
    for it.Next() {
        key := string(it.Key())
        addrCandidate, txid, err := tdpos.ParseNominateRecordsKey(key)
        if err != nil {
            return nil, err
        }
        nominateRecord := &pb.DposNominateInfo{
            Candidate: addrCandidate,
            Txid:      txid,
        }
        nominateRecords = append(nominateRecords, nominateRecord)
    }
    return nominateRecords, nil
}
func (t *consensusReader) GetDposCandidates() ([]string, error) {
    candidates := []string{}
    it := t.Utxovm.ScanWithPrefix([]byte(tdpos.GenCandidateNominatePrefix()))
    defer it.Release()
    for it.Next() {
        key := string(it.Key())
        addr, err := tdpos.ParseCandidateNominateKey(key)
        if err != nil {
            return nil, err
        }
        candidates = append(candidates, addr)
    }
    return candidates, nil
}
