# Configuración de latexmk para el informe.
# Con esto, compilar con «latexmk main.tex» usa LuaLaTeX por defecto.

# 4 = generar PDF con LuaLaTeX
$pdf_mode = 4;

# Motor explícito (redundante con $pdf_mode=4, pero deja constancia)
$lualatex = 'lualatex --interaction=nonstopmode %O %S';

# Biber para la bibliografía (biblatex)
$biber = 'biber %O %S';

# Limpieza extendida: latexmk -c borra también estos archivos
$clean_ext = 'bbl bcf run.xml synctex.gz fls fdb_latexmk nav snm vrb';
