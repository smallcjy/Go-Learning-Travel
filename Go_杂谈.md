## Go语言中的rune
### **Go 语言中的 `rune`**
在 Go 语言中，`rune` 是 `int32` 的别名，**用于表示 Unicode 代码点（Code Point）**。它的作用是方便处理**字符（Character）**，尤其是多字节字符（如中文、日文、韩文等）。

---

## **1. `rune` 的本质**
在 Go 语言中：
- **字符串（string）是 UTF-8 编码的字节序列**，不能直接按照索引访问字符。
- **`rune` 是 `int32` 类型的别名**，用于表示一个 Unicode 代码点，可以正确处理多字节字符。

### **示例**
```go
package main

import "fmt"

func main() {
    var r rune = '你'  // 单引号表示字符
    fmt.Println(r)      // 输出：20320
    fmt.Printf("%c\n", r) // 输出：你
}
```
### **解释**
- `'你'` 是一个 Unicode 字符，它的 **Unicode 代码点是 `U+4F60`，即十进制 `20320`**。
- `fmt.Printf("%c\n", r)` 可以正确打印出 `你`。

---

## **2. `rune` 在字符串中的应用**
Go 语言的字符串是 **UTF-8 编码的字节序列**，索引访问字符串时得到的是**字节（byte）**，而 `rune` 用于正确处理 Unicode 字符。

### **字符串索引 vs `rune` 遍历**
```go
package main

import "fmt"

func main() {
    s := "你好"

    // 直接访问字符串索引，得到的是字节
    fmt.Println("直接访问字符串索引：")
    fmt.Println(s[0], s[1], s[2]) // 228 189 160 (UTF-8 编码)

    // 使用 rune 遍历字符串
    fmt.Println("使用 rune 遍历：")
    for _, r := range s {
        fmt.Printf("%c ", r) // 正确输出：你 好
    }
}
```
### **解释**
1. `s[0], s[1], s[2]` 访问的是 UTF-8 **字节**，而不是字符。
2. `range` 遍历字符串时，会自动解码 UTF-8，将字符转换为 `rune`。

**输出**
```
直接访问字符串索引：
228 189 160
使用 rune 遍历：
你 好
```

---

## **3. `rune` 和 `[]byte` 互相转换**
### **字符串转换为 `rune` 切片**
```go
package main

import "fmt"

func main() {
    s := "Hello, 世界"

    runes := []rune(s) // 将字符串转换为 rune 切片
    fmt.Println(runes) // [72 101 108 108 111 44 32 19990 30028]

    fmt.Println(string(runes)) // 重新转换回字符串：Hello, 世界
}
```
- `[]rune(s)` 可以正确拆分 Unicode 字符，即使字符是多字节的（如 `世`）。
- `string(runes)` 可以将 `rune` 切片重新转换回字符串。

### **字符串转换为 `[]byte`**
```go
package main

import "fmt"

func main() {
    s := "Hello, 世界"

    bytes := []byte(s) // 将字符串转换为字节切片
    fmt.Println(bytes) // [72 101 108 108 111 44 32 228 184 150 231 149 140]
}
```
- `[]byte(s)` 会按照 **UTF-8 编码**拆分字符串，返回字节切片（每个 Unicode 字符可能占多个字节）。
- `rune` 适用于**按字符操作**，`byte` 适用于**按字节操作**。

---

## **4. `rune` 用于字符串修改**
因为 Go 的字符串是**不可变的**，如果需要修改字符串，可以先转换为 `rune` 切片。

```go
package main

import "fmt"

func main() {
    s := "Hello"
    runes := []rune(s)
    runes[0] = 'Y'
    newStr := string(runes)
    fmt.Println(newStr) // "Yello"
}
```
- `string` 不能直接修改，但 `[]rune` 可以修改字符。

---

## **5. `rune` 用于判断字符类型**
`unicode` 包提供了很多实用函数，如判断字符是否是字母、数字等。

```go
package main

import (
    "fmt"
    "unicode"
)

func main() {
    r := '你'
    fmt.Println(unicode.IsLetter(r)) // true
    fmt.Println(unicode.IsDigit(r))  // false
}
```
- `unicode.IsLetter(r)` 判断字符是否是字母。
- `unicode.IsDigit(r)` 判断字符是否是数字。

---

## **6. `rune` 计算字符串长度**
因为字符串是 UTF-8 编码，直接使用 `len(s)` 得到的是字节数，而不是字符数。

```go
package main

import "fmt"

func main() {
    s := "你好世界"
    fmt.Println("字节数:", len(s)) // 12（UTF-8 编码，每个汉字占 3 个字节）
    fmt.Println("字符数:", len([]rune(s))) // 4（正确的字符数）
}
```
- `len(s)` 返回字节数。
- `len([]rune(s))` 返回字符数。

---

## **总结**
| **特性** | **说明** |
|----------|----------|
| `rune` 是 `int32` 别名 | 存储 Unicode 代码点（Code Point） |
| Go 字符串是 UTF-8 | 索引访问返回 `byte`，无法直接获取字符 |
| `[]rune(s)` | 正确拆分字符串中的 Unicode 字符 |
| `range` 遍历字符串 | 自动转换 `rune`，避免乱码 |
| `unicode` 包 | 处理 `rune`（判断字母、数字等） |

---

## Go中strings的FieldFunc

strings.FieldsFunc 是 Go 语言 strings 标准库中的一个函数，用于按照自定义规则切割字符串。它允许使用自定义的分隔逻辑来拆分字符串，而不是像 strings.Split 那样仅使用固定的字符或空格作为分隔符。

1. FieldsFunc 的基本语法

```go
func FieldsFunc(s string, f func(rune) bool) []string
```
- s：要拆分的字符串。
- f：一个回调函数，定义了哪些字符是分隔符。
- 该回调函数的参数是 rune（一个 Unicode 码点）。如果 f(rune) == true，表示这个字符是分隔符，字符串将在该字符处分割。

## strconv.Itoa
strconv.Itoa 是 Go 语言 strconv 标准库中的一个函数，用于将整数转换为字符串。

**基本语法：**
```go
func strconv.Itoa(i int) string
```
- i：要转换的整数。
- 返回值：整数 i 的字符串表示。

## Go中的JSON