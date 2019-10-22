package main

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Factom-Asset-Tokens/factom"
	"github.com/FactomWyomingEntity/private-pool/sharesubmit/minutekeeper"
)

func main() {

	logrus.SetLevel(logrus.TraceLevel)
	cl := factom.NewClient()
	mn := minutekeeper.NewMinuteKeeper(cl)

	mn.Run(context.Background())
}
