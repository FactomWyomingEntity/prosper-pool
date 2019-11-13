package main

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Factom-Asset-Tokens/factom"
	"github.com/FactomWyomingEntity/prosper-pool/minutekeeper"
)

func main() {

	cl := factom.NewClient()
	mn := minutekeeper.NewMinuteKeeper(cl)
	mn.Logger.SetLevel(logrus.TraceLevel)

	mn.Run(context.Background())
}
