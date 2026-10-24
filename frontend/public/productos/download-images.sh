#!/bin/bash
# Descarga imágenes CC0 de cosméticos desde Unsplash (URLs públicas)

cd "$(dirname "$0")"

# Array de URLs Unsplash públicas de cosméticos
declare -a images=(
  "https://images.unsplash.com/photo-1598131143097-8b74394b95c6?w=300&q=80:serum.jpg"
  "https://images.unsplash.com/photo-1556228578-8c89e6adf883?w=300&q=80:crema.jpg"
  "https://images.unsplash.com/photo-1608571423902-eed4a5ad8108?w=300&q=80:tonico.jpg"
  "https://images.unsplash.com/photo-1631730486269-d06b46714d8f?w=300&q=80:mascarilla.jpg"
  "https://images.unsplash.com/photo-1596462502278-af407713fc22?w=300&q=80:limpiador.jpg"
  "https://images.unsplash.com/photo-1617634924626-92b5b8b5e17f?w=300&q=80:toner.jpg"
  "https://images.unsplash.com/photo-1631730486269-d06b46714d8f?w=300&q=80:locion.jpg"
  "https://images.unsplash.com/photo-1615996001375-fd1cb529157d?w=300&q=80:serum-vitamina.jpg"
)

echo "Descargando imágenes de cosméticos..."
for item in "${images[@]}"; do
  url="${item%:*}"
  filename="${item##*:}"
  echo "Descargando: $filename"
  curl -s -o "$filename" "$url" || echo "Error descargando $filename"
done

ls -lh *.jpg
echo "✓ Descarga completada"
