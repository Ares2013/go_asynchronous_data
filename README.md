##### MySql异步同步数据
>项目目录
>   |----bin  
>   |-----|----govendor.exe  
>   |----lib  
>   |----pkg   
>   |----src  
>   |-----|----vendor   
>   |----test  
###### 1.安装包管理工具govendor 
设置GOPATH ,如果以前设置了GOPATH，保持设置一个（我个人经验，设置多个，如果第二个有相同的包就不会安装到新的工作区里）。

##### Use a vendor tool like Govendor  
###### go get govendor
>$ go get github.com/kardianos/govendor
##### Create your project folder and cd inside
>$ mkdir -p $GOPATH/src/github.com/myusername/project && cd "$_"
Vendor init your project and add gin
$ govendor init
$ govendor fetch github.com/gin-gonic/gin@v1.2
Copy a starting template inside your project
$ curl https://raw.githubusercontent.com/gin-gonic/gin/master/examples/basic/main.go > main.go
Run your project
$ go run main.go


#### 其他安装的扩展
以我使用的经验，可以放到vendor中区

  gocode
  gopkgs
  go-outline
  go-symbols
  guru
  gorename
  dlv
  godef
  goreturns
  golint