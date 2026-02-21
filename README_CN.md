# GoDo - Go 开发加速工具

GoDo（Go Development Accelerator Tool）是一个面向 Go Web 项目的 CLI 脚手架与代码生成工具，提供项目初始化、控制器/路由/模型/中间件生成，以及跨平台构建能力，帮助你快速落地一个可运行的 API 项目骨架。

适合日常使用 **Gin + GORM**（以及常见分层：controller/service/model/router）开发 API 的同学，用它可以把重复的目录结构、样板代码和路由维护交给生成器完成。

- 仓库：<https://github.com/jiajia556/godo>
- License：MIT

---

## 功能概览

- `init`：初始化项目结构与 `godoconfig.json`
- `gen cmd`：生成新的 cmd（一个独立可运行的服务入口）
- `gen ctrl`：生成控制器（可附带 actions）
- `gen act`：给已有控制器追加 actions
- `gen rt`：基于控制器注释（AST 分析）生成/更新路由
- `gen model`：从 SQL 文件或配置生成数据库模型
- `gen mdw`：生成 Gin 中间件文件
- `build`：跨平台构建并输出到 `bin/`（构建前会自动生成路由）

---

## 安装

### 方式 1：源码编译

```bash
git clone https://github.com/jiajia556/godo.git
cd godo
go mod download
go build -o godo ./cmd/godo
```

Windows：生成 `godo.exe`，将其放到 PATH 可访问目录。

### 方式 2：go install

```bash
go install github.com/jiajia556/godo/cmd/godo@latest
```

验证：

```bash
godo --version
godo --help
```

---

## 快速开始（推荐流程）

> 下面以生成默认 API 项目为例。

### 1）初始化项目

```bash
godo init myproject
cd myproject
```

### 2）查看初始化后的项目目录

> `init` 会生成一个可运行的默认 API 项目骨架（默认 cmd 一般为 `default-api`）。不同版本模板可能略有差异，但大体结构如下：

```text
myproject/
├── bin/                          # 构建产物输出目录
├── cmd/
│   └── default-api/
│       └── main.go               # 默认 API 入口
├── internal/
│   ├── common/                   # 公共代码（配置/数据库/基础类型等）
│   └── default-api/
│       ├── config/               # 模块配置
│       ├── service/              # 业务服务层
│       └── transport/http/
│           ├── api/controller/   # 控制器（godo gen ctrl 生成到这里）
│           └── router/           # 路由（godo gen rt 生成/更新）
├── godoconfig.json               # GoDo 配置文件
├── go.mod
└── go.sum
```

### 3）生成控制器（可选带 actions）

```bash
# 仅生成控制器骨架
godo gen ctrl user

# 生成控制器 + actions（ActionName:HTTPMethod，HTTPMethod 可省略）
godo gen ctrl user GetList:POST GetDetail:GET Create:POST Update:POST Delete:GET
```

### 4）生成路由（AST 自动分析控制器注释）

```bash
godo gen rt
```

### 5）构建并运行

```bash
# 构建指定 app（cmd 名称），输出到 bin/
godo build default-api

# 运行（Windows 会是 default-api.exe）
./bin/default-api
```

---

## 使用文档（更详细）

### 1）init：初始化项目

```bash
godo init <project-name>
```

约定：
- `<project-name>` 可以是简单目录名（如 `myapp`），也可以是模块路径（如 `example.com/myapp`）。
- 初始化后会生成 `godoconfig.json`，后续命令会优先读取它。

### 2）gen cmd：生成新的 cmd 模块

生成一个新的可运行模块（例如新增一个 `admin-api`）：

```bash
godo gen cmd admin-api
```

生成后通常会新增：
- `cmd/admin-api/main.go`
- `internal/admin-api/...`（对应模块的 controller/router/service 等目录）

### 3）gen ctrl：生成控制器

```bash
godo gen ctrl <controller-route> [actions...]

# 指定生成到哪个 cmd 模块（可选）
godo gen ctrl <controller-route> [actions...] --cmd <cmd-name>
```

参数说明：
- `controller-route`：控制器路由名，支持多级（例如 `user/profile`）。
- `actions`：可选。格式建议用 `ActionName:HTTPMethod`，例如 `GetDetail:GET`。HTTPMethod可省略，默认POST。

示例：

```bash
# 在默认 cmd 下生成 user 控制器
godo gen ctrl user

# 指定生成到 admin-api 模块
godo gen ctrl user GetDetail:GET --cmd admin-api
```

### 4）gen act：给已有控制器追加 actions

```bash
godo gen act [actions...] --ctrl <controller-route>

# 可选：指定 cmd 模块
godo gen act [actions...] --ctrl <controller-route> --cmd <cmd-name>
```

说明：
- `--ctrl`（或 `-c`）必填，用于定位要修改的控制器。
- `actions` 可以是多个（空格分隔）。

示例：

```bash
godo gen act Export:POST Import:POST -c user
```

### 5）gen rt：生成/更新路由

```bash
godo gen rt

# 指定 cmd 模块（可选）
godo gen rt --cmd <cmd-name>
```

它会扫描控制器目录并根据注释生成/更新路由文件。

### 6）gen mdw：生成中间件

```bash
godo gen mdw <middleware-name> [middleware-name...]
```

示例：

```bash
godo gen mdw auth logging
```

### 7）gen model：生成数据库模型

```bash
godo gen model <config.json|schema.sql>
```

说明：
- 该命令只接受 1 个参数：文件路径。
- 你可以传：
  - `schema.sql`：包含 `CREATE TABLE ...` 的 SQL 文件；或
  - `config.json`：数据库连接/生成配置文件（具体字段以项目模板/实现为准）。

示例：

```bash
godo gen model schema.sql
```

### 8）build：构建

```bash
godo build <app-name> [--version <ver>] [--goos <os>] [--goarch <arch>]
```

说明：
- `<app-name>` 是 `cmd/` 下的模块名（例如 `default-api`、`admin-api`）。
- 构建前会自动执行一次路由生成逻辑，确保路由是最新的。

示例：

```bash
# 普通构建
godo build default-api

# 带版本号
godo build default-api --version v1.2.0

# 交叉编译（示例：Linux amd64）
godo build default-api --goos linux --goarch amd64
```

---

## 命令速查

> 以当前实现为准（Cobra CLI）。全局支持：`-v/--verbose`。

```text
godo
├── init  [project-name]
├── gen
│   ├── cmd   [cmd-name]
│   ├── ctrl  [controller-route] [actions...]
│   │        --cmd <name>
│   ├── act   [actions...]
│   │        --cmd <name>
│   │        --ctrl, -c <controller-route>
│   ├── rt
│   │        --cmd <name>
│   ├── model <config.json|schema.sql>
│   └── mdw   [middleware-name...]
└── build [cmd-name]
         --version, -v <ver>
         --goos <os>
         --goarch <arch>
```

说明：
- 多处出现的 `--cmd` 用于指定要操作的命令模块（例如 `default-api`）。为空时会按配置/项目结构推断。
- `build` 会在构建前调用路由生成逻辑（等价于先执行一次路由更新）。

---

## 路由注解（`gen rt` 读取）

在控制器方法注释中使用：

- `@http_method GET|POST`
- `@middleware <name...>`（空格分隔）

示例（只演示注解写法，方法体内容可自行实现）：

```text
// @http_method GET
// @middleware auth logging
// func (ctrl *UserController) GetDetail(c *gin.Context) {
//     // TODO
// }
```

---

## 配置文件：`godoconfig.json`

初始化项目后会生成 `godoconfig.json`。GoDo 在运行时会尝试从当前目录向上查找该文件；如果找不到，会尝试根据 `go.mod` 推断项目根目录信息。

一个最小示例：

```json
{
  "project_name": "myproject",
  "default_cmd": "default-api",
  "default_goos": "linux",
  "default_goarch": "amd64"
}
```

字段含义：
- `project_name`：项目名 / 模块名
- `default_cmd`：默认 cmd 名称（例如 `default-api`）
- `default_goos`：默认构建目标 OS（可被 `build --goos` 覆盖）
- `default_goarch`：默认构建目标架构（可被 `build --goarch` 覆盖）

---

## 常见问题（FAQ）

### 1）提示找不到 godo

- Windows：确认 PATH 后可用 `where godo`
- Linux/macOS：用 `which godo`

### 2）`gen rt` 没有生成到预期路由

- 确认控制器生成位置是否在对应模块的 controller 目录下
- 确认方法注释包含 `@http_method` / `@middleware`（大小写与格式正确）

### 3）构建产物在哪里

- `godo build <app-name>` 会输出到项目的 `bin/` 目录（Windows 会自动带 `.exe` 后缀）

---

## 可选依赖

部分功能可能会用到额外工具：

```bash
# 用于格式化/整理 imports
go install golang.org/x/tools/cmd/goimports@latest
```
