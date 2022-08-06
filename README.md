# DHT - PPCA 2022

Distributed Hash Table (by xjq817

# Report

## Chord

实现了 Chord 协议，并且通过了测试程序

![chord](C:\Users\Lenovo\Desktop\useforgit\DHT-2022\chord.png)

### 实现思路

本质上是个双向链表，用来保证正确性。

在双向链表的基础上储存倍增列表 finger table ，用于降低时间复杂度。

Force Quit 环节中，每个节点维护一定长度的后继序列，在只有部分节点失效的情况下使这个环不会断裂。同时每个节点备份前驱的数据，当前驱失效时启用备份数据。

## Kademlia

实现了 Kademlia 协议，并且通过了测试程序

![kademlia](C:\Users\Lenovo\Desktop\useforgit\DHT-2022\kademlia.png)

其中在 Quit\&Stabilize test 中将点的个数调为 60 ，减少因为退点退的过多而未能及时 republish 的情况

### 实现思路

定义节点间距离为异或值。每个节点拥有 M 个 k-bucket ，第 i 个 k-bucket 储存与该节点距离前缀为 0 的最长长度为 i 的 k 个活跃节点。

每个数据会放在距离它最近的 k 个节点内，应对部分节点失效的情况。这部分可以通过并发的 republish  和 expire 方法来实现。

## Application - BitTorrent

运用命令行的方式实现文件的上传和下载

上传转化为 torrent 文件以及 magnet 种子，可以通过二者之一下载

![app](C:\Users\Lenovo\Desktop\useforgit\DHT-2022\app.png)

### 实现思路

参考了 [BitTorrent](https://blog.jse.li/posts/torrent/#putting-it-all-together)

将文件切成若干 Piece，把每对 (PieceHash,Piece) 放入 DHT 中

根据文件的 PieceHash 列表可以生成 magnet 种子，将 (magnet,文件列表) 也放入DHT中

磁力链接保存的是种子的哈希，因此可以用一个比较简短的磁力链接获取到种子，进而下载文件。