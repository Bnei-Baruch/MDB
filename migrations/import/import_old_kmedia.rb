require 'active_record'
require 'composite_primary_keys'
require 'yaml'
$kmedia_config = YAML::load(File.open('config/database.yml'))['kmedia']
$mdb_config = YAML::load(File.open('config/database.yml'))['mdb']
Dir[File.dirname(__FILE__) + '/models/**/*.rb'].each {|file| require file }

# virual lesson (virtual_lessons) -> collection
# lesson part (containers) -> content unit
# files (file_assets) -> files

puts "Sequence: #{StringTranslation.get_new_sequence}"
StringTranslation.set_translation(nil, 'en', 'kuku')
puts "Sequence: #{StringTranslation.get_new_sequence}"
puts VirtualLesson.count
puts FileAsset.count
puts Container.count
puts "container file_assets: #{Container.last.file_assets.count}"
puts "container description: #{Container.find(99).description('HEB')}"
puts "container description: #{Container.find(18).description('RUS')}"
puts "file_asset description: #{FileAsset.find(283).description('HEB')}"


puts Collection.count
puts ContentUnit.count
puts MDBFile.count

=begin

VirtualLesson.limit(100).each do |vl|
  name = "Morning lesson"
  cl = Collection.new
  cl.set_name('ENG',name)
  vl.containers.each do |con|
    ContentUnit.create(name: con.name, descri)
  end

end
=end