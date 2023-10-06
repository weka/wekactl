package lambdas

type LambdaType string

const LambdaFetchInfo LambdaType = "fetch"
const LambdaScale LambdaType = "scale"
const LambdaTerminate LambdaType = "terminate"
const LambdaJoin LambdaType = "join"
const LambdaTransient LambdaType = "transient"

type LambdaRuntime string

const LambdaRuntimeAl2 LambdaRuntime = "provided.al2"
const LambdaRuntimeGo1x LambdaRuntime = "go1.x"
const LambdaRuntimeDefault LambdaRuntime = LambdaRuntimeAl2

type LambdaArch string

const LambdaArchArm64 LambdaArch = "arm64"
const LambdaArchX86_64 LambdaArch = "x86_64"
const LambdaArchDefault LambdaArch = LambdaArchArm64

const LambdaHandlerName = "bootstrap"
