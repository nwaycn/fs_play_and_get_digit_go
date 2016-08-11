package nway_fs_o_ctl

import (
	"errors"
	"fmt"
	"nway/utils/eventsocket"
	"nway/utils/log"
	. "nway/utils/nway_print"
	"time"
)

//bool结果为是否为终止符，检查一个输入按键
func CheckADtmfEvent(c *eventsocket.Connection, t *time.Timer, EndDtmf string, TwoDtmfTimer int) (string, error, bool) {
	timer1 := time.NewTimer(time.Second * time.Duration(TwoDtmfTimer))
	var nwayerr chan error
	var nwaydtmf chan string
	var nwayenddtmf chan bool
	nwayerr = make(chan error)
	nwaydtmf = make(chan string)
	nwayenddtmf = make(chan bool)
	go func() {
		for t := range timer1.C {
			nwayerr <- errors.New("Wait key expired")
			timer1.Stop()
			Nway_println("CheckADtmfEvent Timeout:", t)
			return
			//return "", errors.New("Max Time not to press a key  ")
		}
	}()

	go func() {
		for {
			ev, err := c.ReadEvent()
			if err != nil {
				logger.Error("Read dtmf failed")
				nwayerr <- err
				//nwayenddtmf <- false
				//return "", err, false
				return
			}
			if ev.Get("Event-Name") == "DTMF" {
				//有按键
				t.Stop()
				timer1.Stop()
				dtmf := ev.Get("Dtmf-Digit")
				if dtmf == EndDtmf {
					//nwayerr <- nil
					nwayenddtmf <- true
					nwaydtmf <- ""
					//return "", nil, true
					return
				} else {
					//nwayerr <- nil
					nwayenddtmf <- false
					nwaydtmf <- dtmf
					//return dtmf, nil, false
					return
				}
			}
		}
	}()
	var (
		err    error
		dtmf   string
		isdtmf bool
	)
	select {
	case err = <-nwayerr:
		return "", err, false
	case isdtmf = <-nwayenddtmf:
		dtmf = <-nwaydtmf
		return dtmf, nil, isdtmf
	}
	//<-timer1.C
	//Nway_println("CheckADtmfEvent Timeout")
	//return "", errors.New("Wait key expired"), false

}

//按一次完整检测dtmf去检查
func CheckDtmfEvent(c *eventsocket.Connection, t *time.Timer, EndDtmf string, MaxDtmf, TwoDtmfTimer int) (string, error) {
	var dtmf string
	for i := 0; i < MaxDtmf; i++ {

		a_dtmf, err, res := CheckADtmfEvent(c, t, EndDtmf, TwoDtmfTimer)
		if err == nil && res == false {
			dtmf += a_dtmf
		} else {
			if err == nil && res {
				break
			} else {

				return "", err
			}
		}
	}
	return dtmf, nil
}

//按MaxFailure去get dtmf
func CheckDtmfEventMaxFailure(c *eventsocket.Connection, t *time.Timer, EndDtmf string, MaxDtmf, MaxTimer, TwoDtmfTimer, MaxFailure int) (string, error) {

	for i := 0; i < MaxFailure; i++ {
		dtmf, err := CheckDtmfEvent(c, t, EndDtmf, MaxDtmf, TwoDtmfTimer)
		//fmt.Println("The Dtmf:", dtmf)
		if err == nil {

			return dtmf, nil
		} else {
			t.Reset(time.Second * time.Duration(MaxTimer))
		}
	}
	c.Execute("exit", "", false)
	return "", errors.New("Mare than Max Failure")
}

//检测最长时间如果没有按键，则raise Max timer错
func CheckDtmfEventMaxTimer(c *eventsocket.Connection, EndDtmf string, MaxDtmf, MaxTimer, TwoDtmfTimer, MaxFailure int) (string, error) {

	timer2 := time.NewTimer(time.Second * time.Duration(MaxTimer))
	var nwayerr chan error
	var nwaydtmf chan string
	nwayerr = make(chan error)
	nwaydtmf = make(chan string)
	go func() {
		for t := range timer2.C {
			nwayerr <- errors.New("Max Time not to press a key  ")
			timer2.Stop()
			Nway_println("CheckDtmfEventMaxTimer Timeout:", t)
			return
			//return "", errors.New("Max Time not to press a key  ")
		}
	}()
	go func() {
		dtmf, err := CheckDtmfEventMaxFailure(c, timer2, EndDtmf, MaxDtmf, MaxTimer, TwoDtmfTimer, MaxFailure)
		if err == nil {
			nwaydtmf <- dtmf
		} else {
			nwayerr <- err
		}
	}()
	var (
		err  error
		dtmf string
	)
	select {
	case err = <-nwayerr:
		return "", err
	case dtmf = <-nwaydtmf:
		return dtmf, nil
	}
	//整个按键超时，提示按键出错

}

/*
//播音取按键
//返回:
//参数:handle:会话handle
//filename:语音文件名称，多个文件以分号";"隔开,文件名称可以带.wav,扩展名,假如没有,默认是.pcm扩展名.可指定路径,假如没有,默认是语音程序的./data/system目录下
//EndDtmf:按键结束条件,比如"#"表示按#号结束输入,""表示没有结束按键条件//支持最大3个结束按键 比如 EndDtmf="*0#" 表示按 0，* 或者#都可以结束
//MaxDtmf:最大按键个数结束条件,0表示没有按键个数结束条件
//MaxTimer:按键最大时间结束条件,单位秒,0表示没有最大时间结束条件
//TwoDtmfTimer:两个按键间的间隔时间结束条件,单位秒,0表示没有两个按键间的间隔时间结束条件
//dtmf:收到的用户的按键(输出参数),包括结束按键的条件,比如"#"
//说明:假如只有播音,不收取按键 设置：MaxDtmf=0

*/
func PlayGetDigits(c *eventsocket.Connection, filename, invalidfile, EndDtmf string, MaxDtmf, MaxTimer, TwoDtmfTimer, MaxFailure int) (string, error) {
	var params, dtmf string
	params = fmt.Sprintf("1 %d %d %d %s %s %s", MaxDtmf, MaxFailure, MaxTimer, EndDtmf, filename, invalidfile)
	//fmt.Println("play_and_get_digits ", params)
	dtmfe, err := c.Execute("play_and_get_digits", params, false)
	if err != nil {
		Nway_println("Get dtmf error:", err)
		return "", err
	} else {
		Nway_println("the dtmf:", dtmfe.Get("Dtmf-Digit"))
		for {
			Nway_println("A new event for dtmf")
			_, err = c.ReadEvent()
			if err != nil {
				Nway_println("Read dtmf failed")
				return "", err
			}
			Nway_println("Check dtmf ......")
			dtmf, err = CheckDtmfEventMaxTimer(c, EndDtmf, MaxDtmf, MaxTimer, TwoDtmfTimer, MaxFailure)
			Nway_println("the dtmf :", dtmf)
			if err != nil {
				return "", err
			} else {
				break
			}
		}

	}
	return dtmf, nil
}
