using System.Diagnostics;
using System.Net;
using System.Text;
using System.Text.Json;
using System.Text.Json.Nodes;
using System.Text.RegularExpressions;
using static MoreLinq.Extensions.BatchExtension;

Console.WriteLine("Hello, World!");

var existingWords = WordGenLib.GridReader.AllowedWords();

//var wikiapi  = @"https://en.wikipedia.org/w/api.php?action=query&format=json&prop=linkshere&redirects=1&formatversion=2&lhprop=title%7Credirect&lhnamespace=0&lhshow=!redirect&lhlimit=25&titles=";
//File.WriteAllLines("../../../wikipedia2.txt",
//    File.ReadLines("../../../wikipedia.txt", Encoding.UTF8)
//    .Where(title =>
//    {
//        HttpWebRequest request = (HttpWebRequest)WebRequest.Create(wikiapi + Uri.EscapeDataString(title));
//        request.AutomaticDecompression = DecompressionMethods.GZip | DecompressionMethods.Deflate;

//        using (HttpWebResponse response = (HttpWebResponse)request.GetResponse())
//        using (Stream stream = response.GetResponseStream())
//        using (StreamReader reader = new StreamReader(stream))
//        {
//            var jsonString = reader.ReadToEnd();
//            var json = JsonNode.Parse(jsonString);
//            var pageNode = json?["query"]?["pages"]?.AsArray()?[0];

//            if (pageNode == null) throw new Exception("JSON page node null in response\n" + jsonString + "\nrequest\n" + title);

//            var links = pageNode?["linkshere"]?.AsArray();
//            if (links == null) return false;
//            return links.Count >= 25;
//        }
//    }));

int gridSize = 10;  // 10 x 10

var generator = WordGenLib.Generator.Create(gridSize);
var grid = generator.GenerateGrid();


for (int i = 0; i < gridSize; ++i)
{
    for (int j = 0; j < gridSize; ++j)
    {
        char cell = grid[i, j] switch
        {
            null => '_',
            char v => v,
        };

        Console.Write(cell);
        Console.Write(' ');
    }
    Console.WriteLine();
}
