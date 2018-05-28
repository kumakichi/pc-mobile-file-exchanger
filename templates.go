package main

const (
	indexTemplate = customHead + `
<!DOCTYPE html>
<head>
   <style>
        .child {
            float: left;
            width: 100%;
            height: 33%;
        }
      body {
          text-align: center;
          font-size:400%;
          margin: auto;
      }
   </style>
</head>
`

	uploadTemplate = customHead + `
<body>
   <form action='/upload' method='post' enctype="multipart/form-data">
      <input id='uploadInput1' class='uniform-file' name='upfile1' type='file'/>
      <input id='uploadInput2' class='uniform-file' name='upfile2' type='file'/>
      <input id='uploadInput3' class='uniform-file' name='upfile3' type='file'/>
      <input id='uploadInput4' class='uniform-file' name='upfile4' type='file'/>
      <input id='uploadInput5' class='uniform-file' name='upfile5' type='file'/>
      <br>
      <br>
      <br>
      <br>
      <input type="submit" value="upload" />
   </form>
</body>`

	upResultTemplate = customHead + `
<body>
   <p style="font-size: 200%;">{{.OkFiles}} was/were uploaded to {{.FilePath}}</p>
   {{ if (ne .FailedFiles "") }}
     <br>
     <br>
     <p style="font-size: 200%;">{{.FailedFiles}} was/were failed to upload</p>
   {{ end }}
</body>`

	customHead = `
<div class="container">
{{ if (ne .ToPC "") }}
  <a href="{{ .ToPC }}" class="child">Upload</a>
{{ else }}
  <a href="{{ .FromPC }}" class="child">Get Files</a>
{{ end }}

{{ if (ne .NoQrcode true) }}
  <a href="{{ .ToQrcode }}" class="child">QR Code</a>
{{ else }}
  <a href="#" class="child">QR Code</a>
{{ end }}

{{ if (ne .Title "Index Page") }}
  <a href="{{ .ToIndex }}" class="child">Index</a>
{{ else }}
  <a href="{{ .ToIndex }}" class="child">Upload</a>
{{ end }}
</div>

<!DOCTYPE html>
<head>
    <title>{{ .Title }}</title>
    <style>
        pre {
            text-align: center;
            font-size: {{ .FontSize }}%;
            margin: auto;
        }
        .container {
            overflow: hidden;
            zoom: 1;
            border: 1px solid red;
        }
        .child {
            float: left;
            width: 33%;
            border: 1px solid greenyellow;
            font-size: {{ .FontSize }}%;
        }
        form{
            text-align: center;
        }
        input {
            font-size: {{ .FontSize }}%;
        }
    </style>
</head>
`
)
