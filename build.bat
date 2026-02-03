@echo off
echo ================================================
echo Building Odoo Backup Tool to EXE...
echo ================================================
echo.

REM Kiem tra Python da cai dat chua
python --version >nul 2>&1
if errorlevel 1 (
    echo ERROR: Python chua duoc cai dat!
    echo Vui long cai dat Python tu https://www.python.org/downloads/
    pause
    exit /b 1
)

echo [1/3] Cai dat cac thu vien can thiet...
pip install -r requirements.txt

echo.
echo [2/3] Build file EXE...
pyinstaller --onefile --name odoo_backup --console odoo_backup.py

echo.
echo [3/3] Sao chep file cau hinh...
copy config.ini dist\config.ini
copy config.ini.example dist\config.ini.example
if exist credentials.json copy credentials.json dist\credentials.json
if exist client_secrets.json copy client_secrets.json dist\client_secrets.json

echo.
echo ================================================
echo BUILD HOAN THANH!
echo ================================================
echo File EXE: dist\odoo_backup.exe
echo.
echo CAC FILE CAN COPY LEN SERVER:
echo   - dist\odoo_backup.exe
echo   - dist\config.ini
echo   - dist\credentials.json (neu co)
echo   - dist\client_secrets.json (neu co)
echo.
pause
