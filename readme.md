<!-- TOC -->

- [1. 说明](#1-说明)
- [2. 网络部分](#2-网络部分)
    - [2.1. proxy](#21-proxy)
    - [2.2. echo](#22-echo)
    - [2.3. benchmark](#23-benchmark)
    - [2.4. chat](#24-chat)
    - [2.5. 其他](#25-其他)
- [3. 底层部分](#3-底层部分)

<!-- /TOC -->

<a id="markdown-1-说明" name="1-说明"></a>
# 1. 说明

一些关于网络/并发的演示代码

<a id="markdown-2-网络部分" name="2-网络部分"></a>
# 2. 网络部分

<a id="markdown-21-proxy" name="21-proxy"></a>
## 2.1. proxy

* [socks(4|4a|5)proxy](socks/readme.md)
* [http(s)proxy](httpproxy/readme.md)
* [proxy_all(proxy集合)](proxy_all/readme.md)
* [tcprelay(代理/慢速/1bit反转)](tcprelay/readme.md)
* [netcat](netcat/readme.md)
* [slowrecv(多路汇聚慢速收)](slowrecv/readme.md)
* [multiplexer(N:1 and 1:n)](multiplexer/readme.md)
* [through_net(中继内网穿透)](multiplexer/readme.md)
* [tls_tun/tls_netcat](tls_tun/readme.md)
* [transparent_proxy(透明代理)](transparent_proxy/readme.md)

<a id="markdown-22-echo" name="22-echo"></a>
## 2.2. echo

* [echo](echo)
* [chargen]()
* [time]()
* [daytime]()
* [discard](discard)
* [ttcp](ttcp/readme.md)
* [simplehttp(s)](simplehttp)

<a id="markdown-23-benchmark" name="23-benchmark"></a>
## 2.3. benchmark

* [pingpong](pingpong/readme.md)
* [httpbench](http_bench/readme.md)
* [zeromq延迟](zeromq/readme.md)

<a id="markdown-24-chat" name="24-chat"></a>
## 2.4. chat

* [hub]()
* [pub-sub]()


<a id="markdown-25-其他" name="25-其他"></a>
## 2.5. 其他

* [incomplete_send(tcp发送数据不完整例子)](incomplete_send/readme.md)
* [file-transfer(文件传输)]()
* [connection-pool(连接池)]()
* [identity-verification(通道身份验证)]()
* [maxconnection(最大连接数)](maxconnection/readme.md)
* [heart-beat(心跳包)]()
* [idleconnection(空闲连接时间轮/堆/升序链表/多线程做法)](idleconnection/readme.md)
* [roundtrip(测时间差)](roundtrip)
* [simple-rpc]()
* [sudoku(数独)]()
* [procmon(简易监控)]()
* [N皇后]()
* [多机求解中位数]()

<a id="markdown-3-底层部分" name="3-底层部分"></a>
# 3. 底层部分

* [自旋锁/读写锁/信号量/互斥量/条件变量](sync/readme.md)
* [lock-free/wait-free](sync/readme.md)
* [blockqueue](blockqueue)
* [mcoroutine(协程实现)]()
* [mtcp(可靠传输协议实现)]()
