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
      <input id='uploadInput' class='uniform-file' name='upfile' type='file'/>
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
   <p>{{.FileName}} was uploaded to {{.FilePath}}</p>
   <a href="{{.FromPC}}">Get Back to files page</a>
   <br>
   <br>
   <a href="{{.ToPC}}">Get Back to upload page</a>
   <br>
   <br>
   <a href="{{.ToIndex}}">Get Back to index page</a>
</body>`
)
