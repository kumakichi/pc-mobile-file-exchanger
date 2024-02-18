package main

const (
	xxqrTemplate = customHead + `
<body>
<img style="display: block;margin-left: auto;margin-right: auto;" src="data:image/png;base64,{{.QrBase}}" alt="QRCode" title="scan this picture to visit"/>
</body>`
	//	`
	//<body>
	//<div>
	//   <p>{{.QrBase}} was/were uploaded to</p>
	//<img src="data:image/png;base64,{{.QrBase}}" alt="QRCode" title="scan this picture to visit"/>
	//</div>
	//</body>
	//`

	uploadTemplate = customHead + `
<body>
   <form action='/upload' method='post' enctype="multipart/form-data">
<input id='uploadInput1' class='uniform-file' name='uploadFile' type='file' multiple/>
      </br>
      </br>
      </br>
      </br>
      <input type="submit" value="upload file[s]" />
   </form>
</body>`

	upResultTemplate = customHead + `
<body>
   <p>{{.OkFiles}} was/were uploaded to {{.FilePath}}</p>
   {{ if (ne .FailedFiles "") }}
     <br>
     <br>
     <p>{{.FailedFiles}} was/were failed to upload</p>
   {{ end }}
</body>`

	customHead = `
<div class="nav">
  <li><a href="{{ .GetFiles }}" class="child">Get Files</a></li>

  <li><a href="{{ .ToQrcode }}" class="child">QR Code</a></li>

  <li><a href="{{ .UploadFiles }}" class="child">Upload</a></li>

  <li><a href="{{ .Clipboard }}" class="child">Clip</a></li>

  <li><a href="../" class="child">../</a></li>
</div>

<!DOCTYPE html>
<head>
    <title>{{ .Title }}</title>
    <style>
        pre {
            text-align: left;
            font-size: {{ .FontSize }}%;
            margin: auto;
        }

        label {
            font-size: {{ .FontSize }}%;
        }

        .nav {
            list-style-type: none;
            margin: 0;
            padding: 0;
            display: flex;
            background-color: silver;
        }

        .nav a {
            text-decoration: none;
            display: block;
            padding: 16px;
            color: white;

      text-align:center;
      border:1px solid #DADADA;
      border-radius:5px;
      cursor:pointer;
      background: linear-gradient(to bottom,#F8F8F8,#27558e);

        }

        .nav a:hover {
            background-color: lightskyblue;
        }

        @media (min-width:800px) {
            .nav {
                justify-content: flex-start;
            }

            li {
                border-left: 1px solid silver;
            }
        }

        @media (min-width:600px) and (max-width:800px) {
            .nav li {
                flex: 1;
            }

            li+li {
                border-left: 1px solid silver;
            }
        }

        @media (max-width: 600px) {
            .nav {
                flex-flow: column wrap;
            }

            li+li {
                border-top: 1px solid silver;
            }
        }


        .child {
            float: left;
            font-size: {{ .FontSize }}%;
        }

        form {
            text-align: center;
        }

        input {
            font-size: {{ .FontSize }}%;
        }
    </style>
</head>
`
)
