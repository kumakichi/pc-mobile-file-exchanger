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
</body>
