package main

import (
    "os"
//    "reflect"
//    "fmt"
    "time"
//    "strconv"
    "encoding/binary"
    "log"
    "os/signal"
    "syscall"
    "periph.io/x/periph/host"
//    "periph.io/x/periph/host/rpi"
    "periph.io/x/periph/conn/i2c/i2creg"
    "periph.io/x/periph/conn/i2c"
)

var dev i2c.Dev
var bus i2c.BusCloser
var addr uint16

const   servo_min = 575
const   servo_max = 2325

type srvo struct {
    reg byte
    ang int
}

    var srvos = map[string]*srvo {
    "pan"  : &srvo{0x01, 0},
    "tilt" : &srvo{0x03, 0},
    }

func delay(ms int) {
    time.Sleep(time.Duration(ms) * time.Millisecond)
}

func inRange(ang int) (res int){
    res = ang
    if ang < -85 {res = -85}
    if ang > 85 {res = 85}
    return res
}

func degToUs(ang int) (us int) {
    ang = inRange(ang)
    ang +=90
//    fmt.Println("degToUs: ang: ", ang)
    s:= servo_max - servo_min
    us = servo_min + int(s / 180.0) * ang
//    fmt.Println("degToUs: us: ", us)
    return us
}

func open(addr uint16) (b i2c.BusCloser, d i2c.Dev) {
// initialise periph
        if _, err := host.Init(); err != nil {
            log.Fatal(err)
    }
        b, err := i2creg.Open("1")
//        fmt.Println("b: ", b, reflect.TypeOf(b))
        if err != nil {
            log.Fatal(err)
    }
//    fmt.Println("opened ok: ", b, reflect.TypeOf(b))
    d = i2c.Dev{b, addr}
//    fmt.Println("d: ", d, reflect.TypeOf(d))
    return b, d
}

func i2cWriteByte(reg byte, data byte) {
    read := make([]byte, 15)
    write := []byte{reg}
    write = append(write, data)
//    fmt.Println("i2cWriteByte: write buffer: ", write)
    if err := dev.Tx(write, read); err != nil {
        log.Fatal(err)
    }
//    fmt.Printf("i2cWriteByte received: %v\n", read)
}

func servo(reg byte, ang int) (res string) {
// load the register
    write := []byte{reg}
//load the angle microsecond pulse data
    x := make([]byte, 2)
    us := degToUs(ang)
    binary.LittleEndian.PutUint16(x, uint16(us))
    write = append(write, x...)
//    fmt.Println("servo: reg, write: ", reg, write)
    
    read := make([]byte, 15)
    if err := dev.Tx(write, read); err != nil {
        log.Fatal(err)
    }
//    fmt.Printf("servo received: %v\n", read)
    delay(250)
    return "done"
    }
    
func servoStop() {
//    fmt.Println("servoStop\n")
    i2cWriteByte(0x00, 0x00)
    delay(250)
}

// note both servos enabled and stopped together

func servoEnable() {
//    fmt.Println("\nservoEnable")
    i2cWriteByte(0x00, 0x03)
    delay(250)
}

func ptHome() (res string) {
//    fmt.Println("\nptHome")
    servo(srvos["pan"].reg,   0)
    srvos["pan"].ang = 0
    servo(srvos["tilt"].reg,  0)
    srvos["tilt"].ang = 0
    return "done"
}

func ptDelta(name string, dlt int) (res string) {
//    fmt.Println("\nptDelta")
    if dlt > 0 {
        for x := srvos[name].ang ; x < srvos[name].ang + dlt ; x++{
            servo(srvos[name].reg, x)
        }
    } else if  dlt < 0 {
        for x := srvos[name].ang ; x > srvos[name].ang + dlt ; x--{
            servo(srvos[name].reg, x)
        }
    }
    srvos[name].ang = srvos[name].ang + dlt
    return "done"
}

func ptDown(cnt int) (res string) {
    for x := 0 ; x < cnt + 1; x++ {
//        fmt.Println("\nptDown")
        srvos["tilt"].ang += 5
        servo(srvos["tilt"].reg, srvos["tilt"].ang)
    }
    return "done"
}

func ptUp(cnt int) (res string) {
    for x := 0 ; x < cnt + 1; x++ {
//        fmt.Println("\nptUp")
        srvos["tilt"].ang -= 5
        servo(srvos["tilt"].reg, srvos["tilt"].ang)
    }
    return "done"
}

func ptRight(cnt int) (res string) {
    for x := 0 ; x < cnt + 1; x++ {
//        fmt.Println("\nptRight")
        srvos["pan"].ang -= 5
        servo(srvos["pan"].reg, srvos["pan"].ang)
    }
    return "done"
}

func ptLeft(cnt int) (res string) {
    for x := 0 ; x < cnt + 1; x++ {
//        fmt.Println("\nptLeft")
        srvos["pan"].ang += 5
        servo(srvos["pan"].reg, srvos["pan"].ang)
    }
    return "done"
}

func main() {
    running := true
// initialise getout
    signalChannel := make(chan os.Signal, 2)
    signal.Notify(signalChannel, os.Interrupt, syscall.SIGTERM)
    go func() {
        sig := <-signalChannel
        switch sig {
        case os.Interrupt:
//            fmt.Println("Stopping on Interrupt")
            running = false
            return
        case syscall.SIGTERM:
//            fmt.Println("Stopping on Terminate")
            running = false
            return
        }
    }()
    
    bus, dev = open(0x15)
    
    servoEnable()
    ptHome()
    delay(2000)
// camera is inverted
// e.g -ve tilt is up
    ptDelta("pan", 20)
    ptDelta("tilt", 20)
    ptDelta("pan", -40)
    ptDelta("tilt", -40)
    ptDelta("pan", 20)
    ptDelta("tilt", 20)
    delay(2000)
    ptHome()
    delay(2000)
    servoStop()
// close the i2c bus
    bus.Close()
}
