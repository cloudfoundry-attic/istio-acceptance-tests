require "sinatra"

count = 0

get "/" do
  count += 1
  if count % 3 == 0
    status 200
    body 'Success!'
  else
    status 500
    body 'Failure!'
  end
end
