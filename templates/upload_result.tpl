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
</body>
