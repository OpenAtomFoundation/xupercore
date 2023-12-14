# 对外暴露的读能力

 // 内核提供
 PreExec(ctx context.Context, in *pb.InvokeRPCRequest) (*pb.InvokeRPCResponse, error) {
 // 内核提供
 PostTx(ctx context.Context, in *pb.TxStatus) (*pb.CommonReply, error) {

 // 服务组合内核提供接口提供
 BatchPostTx(ctx context.Context, in *pb.BatchTxs) (*pb.CommonReply, error) {
 // 服务组合内核提供接口提供
 PreExecWithSelectUTXO(ctx context.Context, in *pb.PreExecWithSelectUTXORequest) (*pb.PreExecWithSelectUTXOResponse, error) {
 // 引擎有提供
 GetBlockChains(ctx context.Context, in *pb.CommonIn) (*pb.BlockChains, error) {

 // 链信息读组件提供
 GetBlockChainStatus(ctx context.Context, in *pb.BCStatus) (*pb.BCStatus, error) {
 ConfirmBlockChainStatus(ctx context.Context, in *pb.BCStatus) (*pb.BCTipStatus, error) {
 GetSystemStatus(ctx context.Context, in *pb.CommonIn) (*pb.SystemsStatusReply, error) {
 GetNetURL(ctx context.Context, in *pb.CommonIn) (*pb.RawUrl, error) {

 // 账本读组件提供 
 QueryTx(ctx context.Context, in *pb.TxStatus) (*pb.TxStatus, error) {
 GetBlock(ctx context.Context, in *pb.BlockID) (*pb.Block, error) {
 GetBlockByHeight(ctx context.Context, in *pb.BlockHeight) (*pb.Block, error) {

 // 合约读组件提供
 QueryContractStatData(ctx context.Context, in *pb.ContractStatDataRequest) (*pb.ContractStatDataResponse, error) {
 GetAccountContracts(ctx context.Context, in *pb.GetAccountContractsRequest) (*pb.GetAccountContractsResponse, error) {
 GetAddressContracts(ctx context.Context, in *pb.AddressContractsRequest) (*pb.AddressContractsResponse, error) {
 GetAccountByAK(ctx context.Context, in *pb.AK2AccountRequest) (*pb.AK2AccountResponse, error) {
 QueryACL(ctx context.Context, in *pb.AclStatus) (*pb.AclStatus, error) {

 // utxo读组件提供
 QueryUtxoRecord(ctx context.Context, in *pb.UtxoRecordDetail) (*pb.UtxoRecordDetail, error) {
 GetBalance(ctx context.Context, in *pb.AddressStatus) (*pb.AddressStatus, error) {
 GetFrozenBalance(ctx context.Context, in *pb.AddressStatus) (*pb.AddressStatus, error) {
 GetBalanceDetail(ctx context.Context, in *pb.AddressBalanceStatus) (*pb.AddressBalanceStatus, error) {
 SelectUTXOBySize(ctx context.Context, in *pb.UtxoInput) (*pb.UtxoOutput, error) {
 SelectUTXO(ctx context.Context, in *pb.UtxoInput) (*pb.UtxoOutput, error) {

 // 共识读组件提供
 DposCandidates(ctx context.Context, in *pb.DposCandidatesRequest) (*pb.DposCandidatesResponse, error) {
 DposNominateRecords(ctx context.Context, in *pb.DposNominateRecordsRequest) (*pb.DposNominateRecordsResponse, error) {
 DposNomineeRecords(ctx context.Context, in *pb.DposNomineeRecordsRequest) (*pb.DposNomineeRecordsResponse, error) {
 DposVoteRecords(ctx context.Context, in *pb.DposVoteRecordsRequest) (*pb.DposVoteRecordsResponse, error) {
 DposVotedRecords(ctx context.Context, in *pb.DposVotedRecordsRequest) (*pb.DposVotedRecordsResponse, error) {
 DposCheckResults(ctx context.Context, in *pb.DposCheckResultsRequest) (*pb.DposCheckResultsResponse, error) {
 DposStatus(ctx context.Context, in *pb.DposStatusRequest) (*pb.DposStatusResponse, error) {
