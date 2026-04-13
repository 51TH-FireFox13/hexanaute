; ============================================================
;  HexaNaute — Script d'installation NSIS
;  Génère : HexaNaute-Setup-v0.5.0.exe
; ============================================================

Unicode True
SetCompressor /SOLID lzma

; --- Infos générales ---
Name              "HexaNaute"
OutFile           "HexaNaute-Setup-v0.5.0.exe"
InstallDir        "$PROGRAMFILES64\HexaNaute"
InstallDirRegKey  HKLM "Software\HexaNaute" "InstallDir"
RequestExecutionLevel admin

; --- Icône de l'installeur ---
Icon              "..\assets\icons\fox.ico"
UninstallIcon     "..\assets\icons\fox.ico"

; --- Infos affichées dans les propriétés ---
VIProductVersion  "0.5.0.0"
VIAddVersionKey   "ProductName"      "HexaNaute"
VIAddVersionKey   "ProductVersion"   "0.5.0"
VIAddVersionKey   "FileDescription"  "Installeur HexaNaute"
VIAddVersionKey   "CompanyName"      "HexaRelay"
VIAddVersionKey   "LegalCopyright"   "Licence MIT"
VIAddVersionKey   "FileVersion"      "0.5.0"

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
!define MUI_FINISHPAGE_RUN          "$INSTDIR\hexanaute.exe"
!define MUI_FINISHPAGE_RUN_TEXT     "Lancer HexaNaute"
!insertmacro MUI_PAGE_FINISH

; Pages de désinstallation
!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES

; Langue
!insertmacro MUI_LANGUAGE "French"

; ============================================================
;  SECTION PRINCIPALE
; ============================================================
Section "HexaNaute" SecMain
    SectionIn RO   ; obligatoire

    SetOutPath "$INSTDIR"

    ; Fichier principal
    File "..\cmd\hexanaute\hexanaute-built.exe"
    Rename "$INSTDIR\hexanaute-built.exe" "$INSTDIR\hexanaute.exe"

    ; Icône (pour les raccourcis)
    SetOutPath "$INSTDIR\assets"
    File "..\assets\icons\fox.ico"

    ; --- Raccourci Bureau ---
    CreateShortcut "$DESKTOP\HexaNaute.lnk" \
        "$INSTDIR\hexanaute.exe" "" \
        "$INSTDIR\assets\fox.ico" 0 \
        SW_SHOWNORMAL "" "Navigateur souverain HexaNaute"

    ; --- Raccourci Menu Démarrer ---
    CreateDirectory "$SMPROGRAMS\HexaNaute"
    CreateShortcut "$SMPROGRAMS\HexaNaute\HexaNaute.lnk" \
        "$INSTDIR\hexanaute.exe" "" \
        "$INSTDIR\assets\fox.ico" 0 \
        SW_SHOWNORMAL "" "Navigateur souverain HexaNaute"
    CreateShortcut "$SMPROGRAMS\HexaNaute\Désinstaller HexaNaute.lnk" \
        "$INSTDIR\Uninstall.exe"

    ; --- Entrée registre (Ajout/Suppression de programmes) ---
    WriteRegStr HKLM "Software\HexaNaute" "InstallDir" "$INSTDIR"
    WriteRegStr HKLM "Software\HexaNaute" "Version"    "0.5.0"

    WriteRegStr HKLM \
        "Software\Microsoft\Windows\CurrentVersion\Uninstall\HexaNaute" \
        "DisplayName"          "HexaNaute"
    WriteRegStr HKLM \
        "Software\Microsoft\Windows\CurrentVersion\Uninstall\HexaNaute" \
        "DisplayVersion"       "0.5.0"
    WriteRegStr HKLM \
        "Software\Microsoft\Windows\CurrentVersion\Uninstall\HexaNaute" \
        "Publisher"            "HexaRelay"
    WriteRegStr HKLM \
        "Software\Microsoft\Windows\CurrentVersion\Uninstall\HexaNaute" \
        "DisplayIcon"          "$INSTDIR\assets\fox.ico"
    WriteRegStr HKLM \
        "Software\Microsoft\Windows\CurrentVersion\Uninstall\HexaNaute" \
        "UninstallString"      '"$INSTDIR\Uninstall.exe"'
    WriteRegStr HKLM \
        "Software\Microsoft\Windows\CurrentVersion\Uninstall\HexaNaute" \
        "QuietUninstallString" '"$INSTDIR\Uninstall.exe" /S'
    WriteRegDWORD HKLM \
        "Software\Microsoft\Windows\CurrentVersion\Uninstall\HexaNaute" \
        "NoModify" 1
    WriteRegDWORD HKLM \
        "Software\Microsoft\Windows\CurrentVersion\Uninstall\HexaNaute" \
        "NoRepair"  1
    WriteRegStr HKLM \
        "Software\Microsoft\Windows\CurrentVersion\Uninstall\HexaNaute" \
        "URLInfoAbout" "https://github.com/51TH-FireFox13/hexanaute"

    ; Créer le désinstalleur
    WriteUninstaller "$INSTDIR\Uninstall.exe"

SectionEnd

; ============================================================
;  SECTION DÉSINSTALLATION
; ============================================================
Section "Uninstall"
    ; Supprimer les fichiers
    Delete "$INSTDIR\hexanaute.exe"
    Delete "$INSTDIR\assets\fox.ico"
    Delete "$INSTDIR\Uninstall.exe"
    RMDir  "$INSTDIR\assets"
    RMDir  "$INSTDIR"

    ; Supprimer les raccourcis
    Delete "$DESKTOP\HexaNaute.lnk"
    Delete "$SMPROGRAMS\HexaNaute\HexaNaute.lnk"
    Delete "$SMPROGRAMS\HexaNaute\Désinstaller HexaNaute.lnk"
    RMDir  "$SMPROGRAMS\HexaNaute"

    ; Supprimer les clés registre
    DeleteRegKey HKLM "Software\HexaNaute"
    DeleteRegKey HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\HexaNaute"

SectionEnd
