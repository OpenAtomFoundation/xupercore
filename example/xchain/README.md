# xchain

通过xchain示例如何基于XuperCore动态内核轻量级实现一条满足场景需要的链。


# TODO

1.state meta没有被初始化。
 
t=2021-01-15T19:21:36+0800 lvl=dbug msg="pack block get max size succ" module=xchain log_id=1610709693_1090551344254216 s_mod=miner call=miner.go:249 pid=9740 sizeLimit=0

2.检查账本和状态机中锁的使用，检查死锁。

比如：MaxTxSizePerBlock 两次加锁，导致死锁。

3.address统一用string。

4.各张表要保证原子性写入，不能多次创建db实例。

5.矿工流程阻塞进程退出&矿工节点进程无法正常退出问题。

6.tdpos CompeteMaster阻塞。
