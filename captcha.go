// 这个验证码包含下面三层元素

// 随机大小和颜色的10个点
// 4位数字的验证码（随机偏转方向、每个点间距随机）
// 一条类似删除线的干扰线

package main

import (
	"bytes"
	crand "crypto/rand"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"encoding/base64"
	"encoding/json"

	"./redisoper"
	"./uuid"
)

type Captcha struct {
	Captchastring string
	Captchatoken  string
}

const (
	stdWidth  = 100 //固定图片宽度
	stdHeight = 40  //固定图片高度
	maxSkew   = 2   //最大偏移量
)

//字体常量信息
const (
	fontWidth  = 5 //字体的宽度
	fontHeight = 8 //字体的高度
	blackChar  = 1
)

// 简化期间使用的字体库
var font = [][]byte{
	{ // 0
		0, 1, 1, 1, 0,
		1, 0, 0, 0, 1,
		1, 0, 0, 0, 1,
		1, 0, 0, 0, 1,
		1, 0, 0, 0, 1,
		1, 0, 0, 0, 1,
		1, 0, 0, 0, 1,
		0, 1, 1, 1, 0,
	},
	{ // 1
		0, 0, 1, 0, 0,
		0, 1, 1, 0, 0,
		1, 0, 1, 0, 0,
		0, 0, 1, 0, 0,
		0, 0, 1, 0, 0,
		0, 0, 1, 0, 0,
		0, 0, 1, 0, 0,
		1, 1, 1, 1, 1,
	},
	{ // 2
		0, 1, 1, 1, 0,
		1, 0, 0, 0, 1,
		0, 0, 0, 0, 1,
		0, 0, 0, 1, 1,
		0, 1, 1, 0, 0,
		1, 0, 0, 0, 0,
		1, 0, 0, 0, 0,
		1, 1, 1, 1, 1,
	},
	{ // 3
		1, 1, 1, 1, 0,
		0, 0, 0, 0, 1,
		0, 0, 0, 1, 0,
		0, 1, 1, 1, 0,
		0, 0, 0, 1, 0,
		0, 0, 0, 0, 1,
		0, 0, 0, 0, 1,
		1, 1, 1, 1, 0,
	},
	{ // 4
		1, 0, 0, 1, 0,
		1, 0, 0, 1, 0,
		1, 0, 0, 1, 0,
		1, 0, 0, 1, 0,
		1, 1, 1, 1, 1,
		0, 0, 0, 1, 0,
		0, 0, 0, 1, 0,
		0, 0, 0, 1, 0,
	},
	{ // 5
		1, 1, 1, 1, 1,
		1, 0, 0, 0, 0,
		1, 0, 0, 0, 0,
		1, 1, 1, 1, 0,
		0, 0, 0, 0, 1,
		0, 0, 0, 0, 1,
		0, 0, 0, 0, 1,
		1, 1, 1, 1, 0,
	},
	{ // 6
		0, 0, 1, 1, 1,
		0, 1, 0, 0, 0,
		1, 0, 0, 0, 0,
		1, 1, 1, 1, 0,
		1, 0, 0, 0, 1,
		1, 0, 0, 0, 1,
		1, 0, 0, 0, 1,
		0, 1, 1, 1, 0,
	},
	{ // 7
		1, 1, 1, 1, 1,
		0, 0, 0, 0, 1,
		0, 0, 0, 0, 1,
		0, 0, 0, 1, 0,
		0, 0, 1, 0, 0,
		0, 1, 0, 0, 0,
		0, 1, 0, 0, 0,
		0, 1, 0, 0, 0,
	},
	{ // 8
		0, 1, 1, 1, 0,
		1, 0, 0, 0, 1,
		1, 0, 0, 0, 1,
		0, 1, 1, 1, 0,
		1, 0, 0, 0, 1,
		1, 0, 0, 0, 1,
		1, 0, 0, 0, 1,
		0, 1, 1, 1, 0,
	},
	{ // 9
		0, 1, 1, 1, 0,
		1, 0, 0, 0, 1,
		1, 0, 0, 0, 1,
		1, 1, 0, 0, 1,
		0, 1, 1, 1, 1,
		0, 0, 0, 0, 1,
		0, 0, 0, 0, 1,
		1, 1, 1, 1, 0,
	},
}

type Image struct {
	*image.NRGBA
	color   *color.NRGBA
	width   int //a digit width  一个数字宽度
	height  int //a digit height  一个数字的高度
	dotsize int
}

func init() {
	//打乱随机种子
	rand.Seed(int64(time.Second))
}

//生成指定长宽的图片
func NewImage(digits []byte, width, height int) *Image {
	img := new(Image)
	r := image.Rect(img.width, img.height, stdWidth, stdHeight) //返回一个矩形Rectangle{Pt(x0, y0), Pt(x1, y1)}。Rectangle代表一个矩形。该矩形包含所有满足Min.X <= X < Max.X且Min.Y <= Y < Max.Y的点。
	img.NRGBA = image.NewNRGBA(r)                               //NewNRGBA函数创建并返回一个具有指定范围的NRGBA,NRGBA类型代表一幅内存中的图像
	img.color = &color.NRGBA{
		uint8(rand.Intn(129)),
		uint8(rand.Intn(129)),
		uint8(rand.Intn(129)),
		0xFF,
	}
	// Draw background (10 random circles of random brightness)
	// 画背景、10个随机亮度的圆
	img.calculateSizes(width, height, len(digits))
	img.fillWithCircles(10, img.dotsize)
	maxx := width - (img.width+img.dotsize)*len(digits) - img.dotsize
	maxy := height - img.height - img.dotsize*2
	x := rnd(img.dotsize*2, maxx)
	y := rnd(img.dotsize*2, maxy)
	// Draw digits. 画验证码
	for _, n := range digits {
		img.drawDigit(font[n], x, y)
		x += img.width + img.dotsize // 下一个验证码字符的起始位置
	}
	// Draw strike-through line. 画类似删除线的干扰线
	img.strikeThrough()
	return img
}

func (img *Image) WriteTo(w io.Writer) (int64, error) {
	return 0, png.Encode(w, img)
}

// 计算几个要显示字符的尺寸，没有开始绘画。
func (img *Image) calculateSizes(width, height, ncount int) {
	// Goal: fit all digits inside the image.
	var border int //边距
	if width > height {
		border = height / 5 // 40/5=8
	} else {
		border = width / 5
	}
	// Convert everything to floats for calculations.转换为浮点数计算
	w := float64(width - border*2)  // 100-8*2=84
	h := float64(height - border*2) // 40-8*2=24
	// fw takes into account 1-dot spacing between digits.
	fw := float64(fontWidth) + 1 // 6
	fh := float64(fontHeight)    // 8
	nc := float64(ncount)        // 4,验证码数字个数
	// Calculate the width of a single digit taking into account only the
	// width of the image.
	nw := w / nc // 84/4=21
	// Calculate the height of a digit from this width.
	nh := nw * fh / fw //  21*8/6 = 28
	// Digit too high?
	if nh > h {
		// Fit digits based on height.
		nh = h            // nh = 24
		nw = fw / fh * nh // 6 / 8 * 24 = 18
	}
	// Calculate dot size.计算点尺寸
	img.dotsize = int(nh / fh) // 24/8 = 3
	// Save everything, making the actual width smaller by 1 dot to account
	// for spacing between digits.
	img.width = int(nw)                // 18
	img.height = int(nh) - img.dotsize // 24-3=21
}

// 随机画指定个数个圆点
func (img *Image) fillWithCircles(n, maxradius int) {
	color := img.color
	maxx := img.Bounds().Max.X
	maxy := img.Bounds().Max.Y
	for i := 0; i < n; i++ {
		setRandomBrightness(color, 255) // 随机颜色亮度
		r := rnd(1, maxradius)          // 随机大小
		img.drawCircle(color, rnd(r, maxx-r), rnd(r, maxy-r), r)
	}
}

// 画 水平线
func (img *Image) drawHorizLine(color color.Color, fromX, toX, y int) {
	// 遍历画每个点
	for x := fromX; x <= toX; x++ {
		img.Set(x, y, color)
	}
}

// 画指定颜色的实心圆
func (img *Image) drawCircle(color color.Color, x, y, radius int) {
	f := 1 - radius
	dfx := 1
	dfy := -2 * radius
	xx := 0
	yy := radius
	img.Set(x, y+radius, color)
	img.Set(x, y-radius, color)
	img.drawHorizLine(color, x-radius, x+radius, y)
	for xx < yy {
		if f >= 0 {
			yy--
			dfy += 2
			f += dfy
		}
		xx++
		dfx += 2
		f += dfx
		img.drawHorizLine(color, x-xx, x+xx, y+yy)
		img.drawHorizLine(color, x-xx, x+xx, y-yy)
		img.drawHorizLine(color, x-yy, x+yy, y+xx)
		img.drawHorizLine(color, x-yy, x+yy, y-xx)
	}
}

// 画一个随机干扰线
func (img *Image) strikeThrough() {
	r := 0
	maxx := img.Bounds().Max.X
	maxy := img.Bounds().Max.Y
	y := rnd(maxy/3, maxy-maxy/3)
	for x := 0; x < maxx; x += r {
		r = rnd(1, img.dotsize/3)
		y += rnd(-img.dotsize/2, img.dotsize/2)
		if y <= 0 || y >= maxy {
			y = rnd(maxy/3, maxy-maxy/3)
		}
		img.drawCircle(img.color, x, y, r)
	}
}

// 画指定的验证码其中一个字符
func (img *Image) drawDigit(digit []byte, x, y int) {
	// 随机偏转方向
	skf := rand.Float64() * float64(rnd(-maxSkew, maxSkew))
	xs := float64(x)
	minr := img.dotsize / 2               // minumum radius
	maxr := img.dotsize/2 + img.dotsize/4 // maximum radius
	y += rnd(-minr, minr)
	for yy := 0; yy < fontHeight; yy++ {
		for xx := 0; xx < fontWidth; xx++ {
			if digit[yy*fontWidth+xx] != blackChar {
				continue
			}
			// Introduce random variations.
			// 引入一些随机变化，不过这里变化量非常小
			or := rnd(minr, maxr)
			ox := x + (xx * img.dotsize) + rnd(0, or/2)
			oy := y + (yy * img.dotsize) + rnd(0, or/2)
			img.drawCircle(img.color, ox, oy, or)
		}
		xs += skf
		x = int(xs)
	}
}

// 设置随机颜色亮度
func setRandomBrightness(c *color.NRGBA, max uint8) {
	minc := min3(c.R, c.G, c.B)
	maxc := max3(c.R, c.G, c.B)
	if maxc > max {
		return
	}
	n := rand.Intn(int(max-maxc)) - int(minc)
	c.R = uint8(int(c.R) + n)
	c.G = uint8(int(c.G) + n)
	c.B = uint8(int(c.B) + n)
}

//三个数中最小的数
func min3(x, y, z uint8) (o uint8) {
	o = x
	if y < o {
		o = y
	}
	if z < o {
		o = z
	}
	return
}

//三个数中最大的数
func max3(x, y, z uint8) (o uint8) {
	o = x
	if y > o {
		o = y
	}
	if z > o {
		o = z
	}
	return
}

// 返回指定范围的随机数
func rnd(from, to int) int {
	//println(to+1-from)
	return rand.Intn(to+1-from) + from
}

const (
	// Standard length of uniuri string to achive ~95 bits of entropy.
	StdLen = 16
	// Length of uniurl string to achive ~119 bits of entropy, closest
	// to what can be losslessly converted to UUIDv4 (122 bits).
	UUIDLen = 20
)

// Standard characters allowed in uniuri string.
// 验证码中标准的字符 大小写与数字
var StdChars = []byte("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789")

// New returns a new random string of the standard length, consisting of
// standard characters.
func New() string {
	return NewLenChars(StdLen, StdChars)
}

// NewLen returns a new random string of the provided length, consisting of
// standard characters.
// 返回指定长度的随机字符串
func NewLen(length int) string {
	return NewLenChars(length, StdChars)
}

// NewLenChars returns a new random string of the provided length, consisting
// of the provided byte slice of allowed characters (maximum 256).
// 返回指定长度，指定候选字符的随机字符串（最大256）
func NewLenChars(length int, chars []byte) string {
	b := make([]byte, length)
	r := make([]byte, length+(length/4)) // storage for random bytes.存储随机数字节，随机字节的存储空间, 多读几个以免
	clen := byte(len(chars))
	maxrb := byte(256 - (256 % len(chars))) // 问题， 为什么要申请这么长的数组？ 看下面循环的 continue 条件
	i := 0
	for {
		// rand.Read() 和 io.ReadFull(rand.Reader) 的区别?
		// http://www.cnblogs.com/ghj1976/p/3435940.html
		if _, err := io.ReadFull(crand.Reader, r); err != nil {
			panic("error reading from random source: " + err.Error())
		}
		for _, c := range r {
			if c >= maxrb {
				// Skip this number to avoid modulo bias.
				// 跳过 maxrb， 以避免麻烦,这也是随机数要多读几个的原因。
				continue
			}
			b[i] = chars[c%clen]
			i++
			if i == length {
				return string(b)
			}
		}
	}
	panic("unreachable")
}

func (img *Image) encodedPNG() []byte {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		fmt.Println(err.Error())
	}
	return buf.Bytes()
}

func pic(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")             //允许访问所有域
	w.Header().Add("Access-Control-Allow-Headers", "Content-Type") //header的类型
	w.Header().Set("content-type", "application/json")
	// 产生验证码byte数组
	d := make([]byte, 4)
	s := NewLen(4)
	ss := ""
	d = []byte(s)
	// 把验证码变成需要显示的字符串
	for v := range d {
		d[v] %= 10
		ss += strconv.FormatInt(int64(d[v]), 32)
	}

	fmt.Println(ss)
	// 图片流方式输出
	//w.Header().Set("Content-Type", "application/json")

	img := NewImage(d, 100, 40)
	imgBuf := img.encodedPNG()
	// fmt.Println(imgBuf)
	//将图片编码为base64
	b4 := base64.StdEncoding.EncodeToString(imgBuf)

	uuid := uuid.GetGuid()
	hooner := &Captcha{b4, uuid}
	//fmt.Println(hooner)
	js, err := json.Marshal(hooner)
	//fmt.Println(*hooner)
	//fmt.Printf("JSON format: %s", js)
	//w.Write(imgBuf)
	//NewImage(d, 100, 40).WriteTo(w)
	w.Write(js)

	//fmt.Println(string(js))

	//fmt.Println(NewImage(d, 100, 40).encodedPNG())
	//打印图片流信息
	//fmt.Println(NewImage(d, 100, 40))

	fmt.Println(uuid)

	//qwe := []byte(uuid)

	//把验证码写入redis
	redis := redisoper.NewRedis("192.168.1.182:6379", "1234")
	res, err := redis.WriteData(uuid, ss, "set")
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Print(res)
	}

	//设置过期时间
	ress, err := redis.Expire(uuid, "1000")
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Print(ress)
	}

}

//验证码验证
func verify(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")             //允许访问所有域
	w.Header().Add("Access-Control-Allow-Headers", "Content-Type") //header的类型
	w.Header().Set("content-type", "application/json")

	r.ParseForm()

	captchastring := r.Form["captchastring"][0]
	captchatoken := r.Form["captchatoken"][0]

	//从redis里面读取数据
	redis := redisoper.NewRedis("192.168.1.182:6379", "1234")
	res, err := redis.GetData(captchatoken, "set")

	if err != nil {
		fmt.Println(err)
	}

	data := res.([]string)

	if captchastring == data[0] {
		mapLit := map[string]string{"state": "1", "msg": "验证成功！！"}
		js, err := json.Marshal(mapLit)
		if err != nil {
			fmt.Println(err)
		} else {
			w.Write(js)
		}
	} else {
		mapLit := map[string]string{"state": "0", "msg": "验证失败！"}
		js, err := json.Marshal(mapLit)
		if err != nil {
			fmt.Println(err)
		} else {
			w.Write(js)
		}
	}
}

func main() {
	http.HandleFunc("/pic", pic)
	http.HandleFunc("/verify", verify)
	s := &http.Server{
		Addr:           ":8080",
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	s.ListenAndServe()
}
