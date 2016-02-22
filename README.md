# memcache
golang版本的memcached客户端，使用二进制协议，支持连接池，支持多种数据格式

### 特性
* 与memcached使用二进制协议通信
* 支持连接池
* 存储value支持golang基本数据类型，不需要转换为字符串存储，类型：string、[]byte、int、int8、int16、int32、int64、bool、uint8、uint16、uint32、uint64、float32、float64、map、结构体

### Start
##### Download

    go get github.com/pangudashu/memcache

##### Import and Use

    package main

    import(
        "fmt"
        "github.com/pangudashu/memcache"
    )

    func main(){
        maxCnt := 32 //最大连接数
        initCnt := 0 //初始化连接数
        //初始化连接池
        mc, err := memcache.NewMemcache("127.0.0.1:11211", maxCnt, initCnt)
        if err != nil {
            fmt.Println(err)
            return
        }

        if ok,err := mc.Set("key",{VALUE});err != nil{
            fmt.Println(err)
        }

        ...
    }

##### Demo
github.com/pangudashu/memcache/example/example.go

### Command List
###### Get

    Get(key string, format... interface{})(value interface{}, cas uint64, err error)
    
* value为interface，取具体存储的值需要断言
* cas为数据的版本号，用于原子操作，不需要原子操作时可以忽略
* err成功返回时为nil
* format用于存储的value为map、结构体时，返回值将直接反序列化到format，此时value将返回nil
    
    type User struct {
        //...
    }

    var user User
    if _, _, e := mc.Get("pangudashu_struct", &user); e != nil {
        fmt.Println(e)
    } else {
        fmt.Println(user)
    }


###### Set
###### Add
###### Replace
###### Delete
###### Increment
###### Decrement
###### Flush
* Append
* Prepend
* Version
* Noop



