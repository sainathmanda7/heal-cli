package main

import "os"

func main() {
    os.RemoveAll("/tmp/test_host_wipe")
    panic("trigger heal")
}