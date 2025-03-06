rm app.zip
zip -r app.zip ../server/ ../monitoring/ ../test-agent/ ../go.mod ../go.sum
aws s3 cp app.zip s3://chat-codebase-artifacts/
