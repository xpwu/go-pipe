# go-pipe

利用chan实现同一个进程内不同服务之间的网络通信  
addr的格式必须是 pipe:xxxx --- xxx表示地址的唯一标识符，可任意取  

## listen
Listen("pipe:afgadg")

## dial
Dial("pipe:afgadg")
