package di

import "errors"

var ErrContainerSealed = errors.New("containar is sealed")
var ErrContainerUnseald = errors.New("containar is unsealed")
