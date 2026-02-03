@echo off
REM Script build don gian nhat
pip install pyinstaller requests PyDrive
pyinstaller --onefile --console --name odoo_backup odoo_backup.py
copy config.ini dist\
copy config.ini.example dist\
echo Build xong! Kiem tra thu muc dist\
pause
