# memcache
golang版本的memcached客户端，使用二进制协议，支持连接池，支持多种数据格式

### 特性
* 与memcached使用二进制协议通信
* 支持连接池
* 存储value支持golang基本数据类型，不需要转换为字符串存储，类型：string、[]byte、int、int8、int16、int32、int64、bool、uint8、uint16、uint32、uint64、float32、float64、map、结构体

### 使用
##### 下载

    go get github.com/pangudashu/memcache

##### 导入

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

##### 示例
github.com/pangudashu/memcache/example/example.go

### 命令列表
###### Get
    
    根据key检索一个元素

    说明：
    Get(key string, format... interface{})(value interface{}, cas uint64, err error)
    
    参数：
    key 要检索的元素的key
    format 用于存储的value为map、结构体时，返回值将直接反序列化到format

    返回值：
    value为interface，取具体存储的值需要断言
    存储的value为map、结构体时,value将返回nil 
    
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
    
    向一个新的key下面增加一个元素

    说明：
    Set(key string, value interface{}, expire ...uint32) (res bool, err error)

    参数：
    key 用于存储值的键名
    value 存储的值，可以为string、[]byte、int、int8、int16、int32、int64、bool、uint8、uint16、uint32、uint64、float32、float64、map、struct等类型
    expire 过期时间，默认0

    返回值：
    设置成功res返回true，err返回nil，否则res返回false，err返回memcache.ErrNotStord

    注意：
    int类型长度与系统位数相关，所以实际存储转为string，建议尽量使用具体长度的类型：int8、int16、int32、int64替换

        //demo
        var value uint32 = 360000000000
        mc.Set("test_value", value, 1800)


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



