@echo off
REM Chatlog 编译脚本
REM 优化点：使用 immutable=1 参数，避免大文件复制

echo ========================================
echo Chatlog 编译脚本
echo ========================================
echo.

REM 设置输出文件名
set OUTPUT=chatlog.exe

REM 清理旧版本
if exist %OUTPUT% (
    echo 清理旧版本...
    del %OUTPUT%
)

REM 编译项目
echo 开始编译...
go build -ldflags="-s -w" -o %OUTPUT% .

REM 检查编译结果
if %ERRORLEVEL% EQU 0 (
    echo.
    echo ========================================
    echo ✓ 编译成功！
    echo ========================================
    echo 输出文件: %OUTPUT%
    
    REM 显示编译前文件大小
    for %%A in (%OUTPUT%) do (
        set ORIGINAL_SIZE=%%~zA
        echo 编译后大小: %%~zA 字节
    )
    
    REM 尝试使用 UPX 压缩
    echo.
    echo 正在压缩可执行文件...
    where upx >nul 2>&1
    if %ERRORLEVEL% EQU 0 (
        upx --best --lzma %OUTPUT%
        if %ERRORLEVEL% EQU 0 (
            echo ✓ UPX 压缩成功！
            for %%A in (%OUTPUT%) do (
                echo 压缩后大小: %%~zA 字节
            )
        ) else (
            echo ✗ UPX 压缩失败，使用未压缩版本
        )
    ) else (
        echo ℹ 未找到 UPX，跳过压缩
        echo   安装 UPX: scoop install upx 或从 https://upx.github.io/ 下载
    )
    
    echo.
) else (
    echo.
    echo ========================================
    echo ✗ 编译失败！
    echo ========================================
    echo 错误代码: %ERRORLEVEL%
    echo.
)

pause
