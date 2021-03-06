package main

import (
	"fmt"
	"github.com/taiyuechain/taiyuechain/common/hexutil"
	taicert "github.com/taiyuechain/taiyuechain/cert"
	"github.com/taiyuechain/taiyuechain/rpc"
	"math/big"
	"os"
	"strconv"
	"sync"
	"time"
)

//send complete
var Count int64 = 0

//Transaction from to account id
var from, to, frequency = 0, 1, 1

//Two transmission intervals
var interval = time.Millisecond * 0

//Send transmission full sleep intervals
var sleep = time.Second

//get all account
var account []string

//time format
var termTimeFormat = "[01-02|15:04:05.000]"

//the pre count
var preCount int64 = 0

//the pre account
var preAccount = ""

//flag sleep
var bSleep = false

//check Account
var CheckAcc = false

var cert []byte

// get par
func main() {
	if len(os.Args) < 8 {
		fmt.Printf("invalid args : %s [count] [frequency] [interval] [sleep] [from] [to] [cert] [\"ip:port\"]\n", os.Args[0])
		return
	}

	count, err := strconv.Atoi(os.Args[1])
	if err != nil {
		fmt.Println(getTime(), "count err")
		return
	}

	frequency, err = strconv.Atoi(os.Args[2])
	if err != nil {
		fmt.Println(getTime(), "frequency err")
		return
	}

	intervalCount, err := strconv.Atoi(os.Args[3])
	if err != nil {
		fmt.Println(getTime(), "interval err")
		return
	} else {
		interval = time.Millisecond * time.Duration(intervalCount)
	}

	sleepCnt, err := strconv.Atoi(os.Args[4])
	if err != nil {
		fmt.Println(getTime(), "sleep err default 10000")
		return
	} else {
		sleep = time.Millisecond * time.Duration(sleepCnt)
	}

	from, err = strconv.Atoi(os.Args[5])
	if err != nil {
		fmt.Println(getTime(), "from err default 0")
	}

	to, err = strconv.Atoi(os.Args[6])
	if err != nil {
		fmt.Println(getTime(), "from err default 1")
	}

	p2p1Byte, err := taicert.ReadPemFileByPath(os.Args[7])
	if err != nil {
		fmt.Println(getTime(), " cert err ", err)
	}
	cert = p2p1Byte

	ip := "127.0.0.1:8545"
	if len(os.Args) == 9 {
		ip = os.Args[8]
	}

	send(count, ip)

}

//get time
func getTime() string {
	return time.Now().Format(termTimeFormat)
}

//send transaction init
func send(count int, ip string) {
	//dial yue
start:
	client, err := rpc.Dial("http://" + ip)
	if err != nil {
		fmt.Println(getTime(), "Dail:", ip, err.Error())
		return
	}

	err = client.Call(&account, "yue_accounts")
	if err != nil {
		fmt.Println(getTime(), "yue_accounts Error", err.Error())
		return
	}
	if len(account) == 0 {
		fmt.Println(getTime(), "no account")
		return
	}
	fmt.Println(getTime(), "account:", account)

	// get balance
	var result string = ""
	err = client.Call(&result, "yue_getBalance", account[from], "latest")
	if err != nil {
		fmt.Println(getTime(), "yue_getBalance Error:", err)
		return
	} else {

		bl, _ := new(big.Int).SetString(result, 10)
		fmt.Println(getTime(), "yue_getBalance Ok:", bl, result)
	}

	//unlock account
	var reBool bool
	err = client.Call(&reBool, "personal_unlockAccount", account[from], "admin", 90000)
	if err != nil {
		fmt.Println(getTime(), "personal_unlockAccount Error:", err.Error())
		return
	} else {
		fmt.Println(getTime(), "personal_unlockAccount Ok", reBool)
	}

	// send
	waitMain := &sync.WaitGroup{}
	for {
		if bSleep {
			bSleep = false
			time.Sleep(sleep)
			client.Close()
			goto start
		}
		waitMain.Add(1)
		go sendTransactions(client, account, count, waitMain)
		frequency -= 1
		if frequency <= 0 {
			break
		}
		time.Sleep(interval)
		// get balance
		err = client.Call(&result, "yue_getBalance", account[from], "latest")
		if err != nil {
			fmt.Println(getTime(), "yue_getBalance Error:", err)
			//return
		} else {
			bl, _ := new(big.Int).SetString(result, 10)
			fmt.Println(getTime(), "yue_getBalance Ok:", bl, result)

			preAccount = result
		}
	}
	waitMain.Wait()
}

//send count transaction
func sendTransactions(client *rpc.Client, account []string, count int, wait *sync.WaitGroup) {
	defer wait.Done()
	waitGroup := &sync.WaitGroup{}

	//发送交易
	for a := 0; a < count; a++ {
		waitGroup.Add(1)
		go sendTransaction(client, account, waitGroup)
	}
	fmt.Println(getTime(), "Send in go Complete", count)
	waitGroup.Wait()
	fmt.Println(getTime(), "Complete", Count)
	if Count > preCount {
		preCount = Count
	} else {
		fmt.Println(getTime(), "tx full sleep")
		bSleep = true
	}
}

//send one transaction
func sendTransaction(client *rpc.Client, account []string, wait *sync.WaitGroup) {
	defer wait.Done()
	map_data := make(map[string]interface{})
	map_data["from"] = account[from]
	map_data["to"] = account[to]
	map_data["value"] = "0x2100"
	map_data["gasPrice"] = "0x5000"
	map_data["cert"] = hexutil.Encode(cert)
	var result string
	err := client.Call(&result, "yue_sendTransaction", map_data)
	if err != nil {
		fmt.Println("sendTransaction ", result, " err ", err)
	}
	if result != "" {
		Count += 1
	}
}
