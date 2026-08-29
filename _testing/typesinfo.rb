#!/usr/bin/env ruby
# encoding: utf-8

RED = "\033[0;31m"
GRN = "\033[0;32m"
NC = "\033[0m"

root = File.expand_path(__dir__)
port = 40000 + (Process.pid % 1000)
addr = "127.0.0.1:#{port}"
server = Process.spawn("gocode", "-s", "-sock", "tcp", "-addr", addr,
                       chdir: root, out: File::NULL, err: File::NULL)
at_exit do
  Process.kill("TERM", server) rescue nil
  Process.wait(server) rescue nil
end

require "socket"
100.times do
  begin
    TCPSocket.new("127.0.0.1", port).close
    break
  rescue Errno::ECONNREFUSED
    sleep 0.01
  end
end

tests = ARGV.empty? ? Dir["#{root}/typesinfo.*"].sort : ARGV.map { |t| File.expand_path(t, root) }
total = ok = 0
tests.each do |dir|
  next unless File.directory?(dir)
  total += 1
  file = File.join(dir, "test.go.in")
  cursor = Dir[File.join(dir, "cursor.*")].first.split(".").last
  expected = File.read(File.join(dir, "out.expected"))
  out = `gocode -sock tcp -addr #{addr} -in #{file} liteide_typesinfo #{file} #{cursor}`
  out = out.gsub(File.expand_path(file), "FILE")
  out = out.gsub(%r{/.*/go\d+\.\d+/src/}, "GOROOT/src/")
  out = out.gsub(%r{GOROOT/src/([^:]+):\d+:\d+}, 'GOROOT/src/\\1:LINE:COLUMN')
  if out == expected
    puts "#{File.basename(dir)}: #{GRN}PASS!#{NC}"
    ok += 1
  else
    puts "#{File.basename(dir)}: #{RED}FAIL!#{NC}"
    puts "Got:\n#{out}Expected:\n#{expected}"
  end
end
puts "\nSummary (total: #{total})"
puts "#{GRN}  PASS#{NC}: #{ok}"
puts "#{RED}  FAIL#{NC}: #{total - ok}"
exit(total == ok ? 0 : 1)
