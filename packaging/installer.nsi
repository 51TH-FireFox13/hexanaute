; ============================================================
;  Fox Browser — Script d'installation NSIS
;  Génère : FoxBrowser-Setup-v0.3.0.exe
; ============================================================

Unicode True
SetCompressor /SOLID lzma

; --- Infos générales ---
Name              "Fox Browser"
OutFile           "FoxBrowser-Setup-v0.3.0.exe"
InstallDir        "$PROGRAMFILES64\FoxBrowser"
InstallDirRegKey  HKLM "Software\FoxBrowser" "InstallDir"
RequestExecutionLevel admin

; --- Icône de l'installeur ---
Icon              "..\assets\icons\fox.ico"
UninstallIcon     "..\assets\icons\fox.ico"

; --- Infos affichées dans les propriétés ---
VIProductVersion  "0.3.0.0"
VIAddVersionKey   "ProductName"      "Fox Browser"
VIAddVersionKey   "ProductVersion"   "0.3.0"
VIAddVersionKey   "FileDescription"  "Installeur Fox Browser"
VIAddVersionKey   "CompanyName"      "Fox Browser Project"
VIAddVersionKey   "LegalCopyright"   "Licence MIT"
VIAddVersionKey   "FileVersion"      "0.3.0"

; --- Interface moderne ---
!include "MUI2.nsh"

!define MUI_ICON              "..\assets\icons\fox.ico"
!define MUI_UNICON            "..\assets\icons\fox.ico"
!define MUI_WELCOMEFINISHPAGE_BITMAP_NOSTRETCH
!define MUI_ABORTWARNING

; Pages d'installation
!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_LICENSE      "..\LICENSE"
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!define MUI_FINISHPAGE_RUN          "$INSTDIR\fox-gui.exe"
!define MUI_FINISHPAGE_RUN_TEXT     "Lancer Fox Browser"
!insertmacro MUI_PAGE_FINISH

; Pages de désinstallation
!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES

; Langue
!insertmacro MUI_LANGUAGE "French"

; ============================================================
;  SECTION PRINCIPALE
; ============================================================
Section "Fox Browser" SecMain
    SectionIn RO   ; obligatoire

    SetOutPath "$INSTDIR"

    ; Fichier principal
    File "..\cmd\fox-gui\fox-gui-built.exe"
    Rename "$INSTDIR\fox-gui-built.exe" "$INSTDIR\fox-gui.exe"

    ; Icône (pour les raccourcis)
    SetOutPath "$INSTDIR\assets"
    File "..\assets\icons\fox.ico"

    ; --- Raccourci Bureau ---
    CreateShortcut "$DESKTOP\Fox Browser.lnk" \
        "$INSTDIR\fox-gui.exe" "" \
        "$INSTDIR\assets\fox.ico" 0 \
        SW_SHOWNORMAL "" "Navigateur souverain Fox Browser"

    ; --- Raccourci Menu Démarrer ---
    CreateDirectory "$SMPROGRAMS\Fox Browser"
    CreateShortcut "$SMPROGRAMS\Fox Browser\Fox Browser.lnk" \
        "$INSTDIR\fox-gui.exe" "" \
        "$INSTDIR\assets\fox.ico" 0 \
        SW_SHOWNORMAL "" "Navigateur souverain Fox Browser"
    CreateShortcut "$SMPROGRAMS\Fox Browser\Désinstaller Fox Browser.lnk" \
        "$INSTDIR\Uninstall.exe"

    ; --- Entrée registre (Ajout/Suppression de programmes) ---
    WriteRegStr HKLM "Software\FoxBrowser" "InstallDir" "$INSTDIR"
    WriteRegStr HKLM "Software\FoxBrowser" "Version"    "0.3.0"

    WriteRegStr HKLM \
        "Software\Microsoft\Windows\CurrentVersion\Uninstall\FoxBrowser" \
        "DisplayName"          "Fox Browser"
    WriteRegStr HKLM \
        "Software\Microsoft\Windows\CurrentVersion\Uninstall\FoxBrowser" \
        "DisplayVersion"       "0.3.0"
    WriteRegStr HKLM \
        "Software\Microsoft\Windows\CurrentVersion\Uninstall\FoxBrowser" \
        "Publisher"            "Fox Browser Project"
    WriteRegStr HKLM \
        "Software\Microsoft\Windows\CurrentVersion\Uninstall\FoxBrowser" \
        "DisplayIcon"          "$INSTDIR\assets\fox.ico"
    WriteRegStr HKLM \
        "Software\Microsoft\Windows\CurrentVersion\Uninstall\FoxBrowser" \
        "UninstallString"      '"$INSTDIR\Uninstall.exe"'
    WriteRegStr HKLM \
        "Software\Microsoft\Windows\CurrentVersion\Uninstall\FoxBrowser" \
        "QuietUninstallString" '"$INSTDIR\Uninstall.exe" /S'
    WriteRegDWORD HKLM \
        "Software\Microsoft\Windows\CurrentVersion\Uninstall\FoxBrowser" \
        "NoModify" 1
    WriteRegDWORD HKLM \
        "Software\Microsoft\Windows\CurrentVersion\Uninstall\FoxBrowser" \
        "NoRepair"  1
    WriteRegStr HKLM \
        "Software\Microsoft\Windows\CurrentVersion\Uninstall\FoxBrowser" \
        "URLInfoAbout" "https://github.com/51TH-FireFox13/fox-browser"

    ; Créer le désinstalleur
    WriteUninstaller "$INSTDIR\Uninstall.exe"

SectionEnd

; ============================================================
;  SECTION DÉSINSTALLATION
; ============================================================
Section "Uninstall"
    ; Supprimer les fichiers
    Delete "$INSTDIR\fox-gui.exe"
    Delete "$INSTDIR\assets\fox.ico"
    Delete "$INSTDIR\Uninstall.exe"
    RMDir  "$INSTDIR\assets"
    RMDir  "$INSTDIR"

    ; Supprimer les raccourcis
    Delete "$DESKTOP\Fox Browser.lnk"
    Delete "$SMPROGRAMS\Fox Browser\Fox Browser.lnk"
    Delete "$SMPROGRAMS\Fox Browser\Désinstaller Fox Browser.lnk"
    RMDir  "$SMPROGRAMS\Fox Browser"

    ; Supprimer les clés registre
    DeleteRegKey HKLM "Software\FoxBrowser"
    DeleteRegKey HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\FoxBrowser"

SectionEnd
