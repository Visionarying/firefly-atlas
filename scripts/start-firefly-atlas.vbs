Option Explicit
' Start the hidden cmd launcher so double-clicking the shortcut does not flash a
' console window. The actual server is opened in its own visible console by the
' cmd script, which keeps startup errors readable.
Dim fso, shell, root, script
Set fso = CreateObject("Scripting.FileSystemObject")
Set shell = CreateObject("WScript.Shell")
root = fso.GetParentFolderName(fso.GetParentFolderName(WScript.ScriptFullName))
script = fso.BuildPath(root, "scripts\start-firefly-atlas.cmd")
shell.Run """" & script & """", 0, False
