# consensus-PBFT

使用Go语言实现面向工业物联网的区块链高效拜占庭容错共识算法

# 上手指南

以下指南将帮助你在本地机器上安装`Goland`和运行拜占庭容错类共识算法（包括`PBFT`和`CD-PBFT`）。关于如何将共识算法部署到`Goland`在线环境，请参考部署小节。

# 安装要求

必须安装`Goland`



# 部署

## 1.对某一个包含有main函数的文件分开编译

```shell
go build xx.go

example

go build .\nodeServer.go

```

## 2.直接运行某一个包含main函数的go 文件[参数该怎么带怎么带]

```shell
go run XX.go

example

go run clientServer.go
```

## 3.如果是某一个项目

```
go build 直接编译
```

生成的可执行文件如果没做特殊处理，那么就是`[项目名.exe]`

## 4.如果需要下载对应的库[我们举一个比较简单的例子]

* 第一步：创建项目`main`函数 `main.go`在项目的顶级目录

* 第二步：初始化项目

  ```shell
  go mod init 项目名 
  ```

   example：我们这里的项目名是`consensusPBFT2`我们执行`go mod init consensusPBFT2`，`consensusPBFT1`同理。

  这个时候我们就会生成我们项目中需要的`go.mod`这里就是我们所需要的第三方依赖包

* 第三步：` go mod tidy` 下载我们项目需要的第3方包这时候会有一个`go.sum`的文件出现列举了下载了详细的东西

* 第四步： 如果你要给别人使用，对方没有`go`环境，或者安装了`go`，没有离线包，那么你则需要打包给别人，这里需要执行`go mod vendor`

  这里就会把离线包打到项目中

  拷贝给别人，别人不需要下载依赖就也可以跑了

# 贡献者

李凤岐，宋晴晴，徐辉，杜学峰，高嘉隆，佟宁，王德广

# 鸣谢

感谢李凤岐老师、佟宁老师、王德广老师的细心指导和耐心陪伴。

感谢男友的全程支持和鼓励。
