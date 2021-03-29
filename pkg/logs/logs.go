package logs

import (
	see "github.com/cihub/seelog"
	"log"
	"os"
)

var Logger see.LoggerInterface

func init() {

	common := "pkg/logs/common.log"
	critical := "pkg/logs/critical.log"
	errorLog := "pkg/logs/error.log"
	if _, err := os.Stat(common); !os.IsNotExist(err) {
		// Do nothing
	}

	file, err := os.Create(common)
	if err != nil {
		log.Println(err)
		return
	}
	defer func() {
		err = file.Close()
		if err != nil {
			log.Println(err)
			return
		}
	}()
	if _, err := os.Stat(critical); !os.IsNotExist(err) {
		// Do nothing
	}

	file, err = os.Create(critical)
	if err != nil {
		log.Println(err)
		return
	}
	defer func() {
		err = file.Close()
		if err != nil {
			log.Println(err)
			return
		}
	}()

	if _, err := os.Stat(errorLog); !os.IsNotExist(err) {
		// Do nothing
	}

	file, err = os.Create(errorLog)
	if err != nil {
		log.Println(err)
		return
	}
	defer func() {
		err = file.Close()
		if err != nil {
			log.Println(err)
			return
		}
	}()

	Logger = see.Disabled
	loadAppConfig()
}

func loadAppConfig() {

	appConfig := `
<seelog type="sync">
	<outputs>
		<filter levels="trace">
			<console formatid="plain"/>
			<file path="./pkg/logs/common.log"/>
		</filter>

		<filter levels="info">
			<console formatid="plain"/>
			<file path="./pkg/logs/common.log"/>
		</filter>

		<filter levels="warn">
			<console formatid="plain" />
			<file path="./pkg/logs/common.log"/>
		</filter>

		<filter levels="error">
			<console formatid="error"/>
			<file path="./pkg/logs/error.log"/>
		</filter>
		
		<filter levels="critical">
			<console formatid="critical"/>
			<file path="./pkg/logs/critical.log"/>
			<smtp formatid="criticalemail" 
				senderaddress="shiftrgh@gmail.com" 
				sendername="Scheduler Microservice" 
				hostname="smtp.gmail.com" 
				hostport="587" 
				username="shiftrgh@gmail.com" 
				password="yoforreal.com">
				<recipient address="shiftrgh@gmail.com"/>
			</smtp>
		</filter>
		
	</outputs>
	<formats>
		<format id="plain" format="%Date/%Time %EscM(46)[%LEVEL]%EscM(49) %Msg%n%EscM(0)" />
		<format id="error" format="%Date/%Time [%LEVEL] %RelFile %Func %Line %Msg%n" />
		<format id="critical" format="%Date/%Time [%LEVEL] %RelFile %Func %Line %Msg%n" />
		<format id="criticalemail" format="Critical error on our server! %n%n%Time %Date [%LEVEL] %FullPath %n%RelFile %n%File  %n%Func %n%Msg%n %nSent by PostIt Scheduler Micro-Service"/>
	</formats>
</seelog>
`

	logger, err := see.LoggerFromConfigAsBytes([]byte(appConfig))
	if err != nil {
		log.Println(err)
		return
	}
	//err = see.ReplaceLogger(logger)
	//if err != nil {
	//	panic(err)
	//	return
	//}

	UseLog(logger)
}

func UseLog(log see.LoggerInterface) {
	Logger = log
}
