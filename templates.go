package main

const (
	indexTemplate = `
<!DOCTYPE html>
<head>
   <title>Choose</title>
   <style>
      body {
          text-align: center;
          font-size:400%;
          margin: auto;
      }
   </style>
</head>
<body>
   <a href="{{.FromPC}}">Get file from PC</a>
   <br>
   <br>
   <a href="{{.ToPC}}">Upload file to PC</a>
</body>`

	uploadTemplate = `
<!DOCTYPE html>
<head>
   <title>Upload file</title>
   <style>
      form{
          text-align: center;
      }
      input {
          font-size:300%;
      }
   </style>
</head>
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

	upResultTemplate = `
<!DOCTYPE html>
<head>
   <title>Upload succeed</title>
   <style>
      body {
          text-align: center;
          font-size:400%;
          margin: auto;
      }
   </style>
</head>
<body>
   <p>{{.OkFiles}} was/were uploaded to {{.FilePath}}</p>
   {{ if (ne .FailedFiles "") }}
   <p>{{.FailedFiles}} was/were failed to upload</p>
   {{ end }}
   <a href="{{.FromPC}}">Get Back to files page</a>
   <br>
   <br>
   <a href="{{.ToPC}}">Get Back to upload page</a>
   <br>
   <br>
   <a href="{{.ToIndex}}">Get Back to index page</a>
</body>`

	customFSHead = `
<!DOCTYPE html>
<head>
    <title>Choose</title>
    <style>
        pre {
            text-align: center;
            font-size:300%;
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
        }
    </style>
</head>
<body>
`

	customFSTail = `
</body>
`
)
