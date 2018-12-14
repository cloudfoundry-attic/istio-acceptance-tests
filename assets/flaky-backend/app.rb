require "sinatra"

count = 0

get "/" do
  count += 1
  if count % 3 == 0
    status 200
    body 'Success!'
  else
    status 503
    body 'Failure!'
  end
end
