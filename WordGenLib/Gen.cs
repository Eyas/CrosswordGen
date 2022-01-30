using System.Collections.Immutable;
using System.Text;

namespace WordGenLib
{
    public class CharSet
    {
        private readonly bool[] _available;
        private readonly char _min;

        public CharSet(char min, char max)
        {
            _min = min;
            _available = new bool[1 + (max - min)];
        }

        public CharSet() : this('`', 'z') { }

        public void Add(char c)
        {
            _available[ c - _min ] = true;
        }

        public bool Contains(char c)
        {
            return _available[c - _min ];
        }
    }
    public class GridDictionary<T> where T : notnull, new()
    {
        private readonly T?[,] _values;
        public GridDictionary(int size)
        {
            _values = new T?[size, size];
        }

        public T GetOrAddDefault((int x, int y) kv)
        {
            var x = _values[kv.x, kv.y];
            if (x == null)
            {
                x = new();
                _values[kv.x, kv.y] = x;
            }
            return x;
        }

        public T this[(int x, int y) kv]
        {
            get
            {
                var x = _values[kv.x, kv.y];
                if (x != null) return x;

                throw new NullReferenceException($"No value found at {kv}");
            }
        }

    }

    public class Generator
    {
        private readonly int gridSize;
        private readonly ImmutableArray<string> commonWords;
        private readonly ImmutableArray<string> commonAndObscureWords;

        private readonly Dictionary<int, ImmutableArray<string>> commonWordsByLength;
        private readonly Dictionary<int, ImmutableArray<string>> obscureWordsByLength;
        private readonly ImmutableArray<ImmutableDictionary<char, ImmutableSortedSet<string>>> wordsByLetterPositon;

        public int GridSize => gridSize;

        public static Generator Create(int gridSize)
        {
            return new(
                gridSize,
                GridReader.COMMON_WORDS
                    .Where(s => s.Length > 2 && s.Length <= gridSize)
                    .ToImmutableArray(),
                GridReader.ALL_WORDS
                    .Where(s => s.Length > 2 && s.Length <= gridSize)
                    .ToImmutableArray()
                );
        }

        internal Generator(int gridSize, ImmutableArray<string> commonWords, ImmutableArray<string> obscureWords)
        {
            this.gridSize = gridSize;
            this.commonWords = commonWords.Where(w => w.Length <= gridSize).ToImmutableArray();
            commonAndObscureWords = commonWords.AddRange(obscureWords.Where(w => w.Length <= gridSize));

            commonWordsByLength = this.commonWords.GroupBy(w => w.Length).ToDictionary(g => g.Key, g => g.ToImmutableArray ());
            obscureWordsByLength = obscureWords.Where(w => w.Length <= gridSize).GroupBy(w => w.Length).ToDictionary(g => g.Key, g => g.ToImmutableArray());

            var x = ImmutableArray.CreateBuilder<ImmutableDictionary<char, ImmutableSortedSet<string>>>(gridSize);
            for (int i = 0; i < gridSize; i++)
            {
                x.Add(commonAndObscureWords.Where(x => x.Length > i).Select(x => (charAt: x[i], word: x)).GroupBy(x => x.charAt, x => x.word).ToImmutableDictionary(group => group.Key, group => group.ToImmutableSortedSet()));
            }
            wordsByLetterPositon = x.ToImmutableArray();
        }

        private record class ScoredLine(string Line, int MaxLength, int NumLetters, int NumLettersOfCommonWords) { }
        private record GridState(ImmutableArray<ImmutableArray<ScoredLine>> Down, ImmutableArray<ImmutableArray<ScoredLine>> Across) {
            public IEnumerable<int> UndecidedDown
                => Down.Select((options, index) => (options, index)).Where(oi => oi.options.Length > 1).Select(oi => oi.index);
            public IEnumerable<int> UndecidedAcross
                => Across.Select((options, index) => (options, index)).Where(oi => oi.options.Length > 1).Select(oi => oi.index);
            public string Key => string.Join(",", Down.Select(opts => opts.IsEmpty ? "[]" : $"[{opts[0]}/{opts.Length}/{opts[^1]}]")) + "///" +
                                string.Join(",", Across.Select(opts => opts.IsEmpty ? "[]" : $"[{opts[0]}/{opts.Length}/{opts[^1]}]"));
        }
        private record class FinalGrid(ImmutableArray<ScoredLine> Down, ImmutableArray<ScoredLine> Across) { }

        private IEnumerable<ScoredLine> AllPossibleLines(int maxLength)
        {
            if (maxLength > gridSize) throw new Exception($"{nameof(maxLength)} ({maxLength}) cannot be greater than {nameof(gridSize)} {gridSize}");
            if (maxLength < 3) yield break;

            foreach (string word in commonWordsByLength[maxLength])
            {
                yield return new ScoredLine(word, maxLength, maxLength, maxLength);
            }

            foreach (string word in obscureWordsByLength[maxLength])
            {
                yield return new ScoredLine(word, maxLength, maxLength, 0);
            }

            // recurse into *[ANYTHING], and [ANYTHING]*
            foreach (var line in AllPossibleLines(maxLength - 1))
            {
                yield return line with { Line = GenHelper.BLOCKED + line.Line };
                yield return line with { Line = line.Line + GenHelper.BLOCKED };
            }

            // recurse into all combination of [ANYTHING]*[ANYTHING]
            for (int i = 3; i < gridSize - 3; ++i)
            {
                int firstLength = i;  // Always >= 3.
                int secondLength = gridSize - (i + 1);  // Always >= 3.

                if (secondLength < 3) throw new Exception($"{nameof(secondLength)} is {secondLength} (i = {i}).");

                foreach (var firstHalf in AllPossibleLines(firstLength))
                {
                    foreach (var secondHalf in AllPossibleLines(secondLength))
                    {
                        yield return new ScoredLine(
                            firstHalf.Line + GenHelper.BLOCKED + secondHalf.Line,
                            Math.Max(firstHalf.MaxLength, secondHalf.MaxLength),
                            firstHalf.NumLetters + secondHalf.NumLetters,
                            firstHalf.NumLettersOfCommonWords + secondHalf.NumLettersOfCommonWords
                            );
                    }
                }
            }
        }

        static GridState Prefilter(GridState state, Direction direction)
        {
            var toFilter = direction == Direction.Horizontal ? state.Across : state.Down;
            var constraint = direction == Direction.Horizontal ? state.Down : state.Across;

            if (toFilter.Any(r => r.Length == 0) || constraint.Any(c => c.Length == 0)) return state;

            // x and y here are abstracted wlog based on toFilter/constraint, not truly
            // connected to Horizontal vs Vertical.
            GridDictionary<CharSet> available = new(state.Across.Length);
            for (int x = 0; x < constraint.Length; x++)
            {
                var c = constraint[x];
                
                foreach (ScoredLine line in c)
                {
                    var arr = line.Line.ToCharArray();
                    for (int y = 0; y < arr.Length; y++)
                    {
                        var chars = available.GetOrAddDefault((x, y));
                        chars.Add(arr[y]);
                    }
                }
            }

            var filtered = toFilter.Select((possibles, y) => possibles.Where(line => {
                for (int x = 0; x < line.Line.Length; ++x)
                {
                    if (!available[(x, y)].Contains(line.Line[x])) return false;
                }
                return true;
            }).ToImmutableArray()).ToImmutableArray();

            if (direction == Direction.Horizontal)
            {
                return state with { Across = filtered };
            }
            else return state with { Down = filtered };
        }

        private static IEnumerable<FinalGrid> AllPossibleGrids(GridState root)
        {
            // If we are at a point in our tree some row/column is unfillable, prune this tree.
            if (root.Down.Any(options => options.Length == 0)) yield break;
            if (root.Across.Any(options => options.Length == 0)) yield break;

            // Prefilter
            {
                int tries = 0;
                Direction direction = Direction.Horizontal;
                while (tries < 4)
                {
                    ++tries;
                    GridState newState = Prefilter(root, direction);
                    if (!Changed(root, newState) && tries > 1) break;

                    root = newState;
                    direction = direction == Direction.Vertical ? Direction.Horizontal : Direction.Vertical;
                }

                // If we are at a point in our tree some row/column is unfillable, prune this tree.
                if (root.Down.Any(options => options.Length == 0)) yield break;
                if (root.Across.Any(options => options.Length == 0)) yield break;
            }

            ImmutableArray<int> undecidedDown = root.UndecidedDown.ToImmutableArray();
            ImmutableArray<int> undecidedAcross = root.UndecidedAcross.ToImmutableArray();

            if (undecidedDown.IsEmpty && undecidedAcross.IsEmpty)
            {
                yield return new FinalGrid(
                    Down: root.Down.Select(col => col[0]).ToImmutableArray(),
                    Across: root.Across.Select(row => row[0]).ToImmutableArray()
                );
                yield break;
            }

            var dirs = new[] { Direction.Horizontal, Direction.Vertical };
            var parity = Random.Shared.Next(2);

            for (int i = 0; i < dirs.Length; ++i)
            {
                var dir = dirs[(parity + i) % 2];
                var undecided = dir == Direction.Horizontal ? undecidedAcross : undecidedDown;
                if (undecided.IsEmpty) continue;
                foreach (var final in AllPossibleGrids(root, undecided, dir)) yield return final;

                // After the first successful "search" down the tree, we're done here. The second
                // will be a mirror image.
                yield break;
            }
        }

        private static IEnumerable<FinalGrid> AllPossibleGrids(GridState root, ImmutableArray<int> undecided, Direction dir)
        {
            var optionAxis = (dir == Direction.Horizontal ? root.Across : root.Down);
            var oppositeAxis = (dir == Direction.Horizontal ? root.Down : root.Across);

            // Trim situations where horizontal and vertal words are same.
            for (int i = 0; i < optionAxis.Length; ++i)
            {
                if (optionAxis[i].Length > 1) continue;
                if (oppositeAxis[i].Length > 1) continue;

                if (optionAxis[i][0].Line == oppositeAxis[i][0].Line) yield break;
            }

            foreach (int index in undecided)
            {
                var options = optionAxis[index];

                // The below loop "makes decisions" and recurses. If we already
                // have one attempt, that means it's already pre-decided.
                if (options.Length == 1) continue;

                foreach (var attempt in options)
                {
                    var attemptOpposite = oppositeAxis.ToArray();

                    for (int i = 0; i < attempt.Line.Length; i++)
                    {
                        // WLOG say we dir is Horizontal, and opopsite is Vertical.
                        // we have:
                        //
                        // W O R D
                        // _ _ _ _
                        // _ _ _ _
                        // _ _ _ _
                        //
                        // Then go over each COL (i), filtering s.t. possible lines
                        // only include cases where col[i]'s |attempt|th character == attempt[i].
                        var constriant = attempt.Line[i];

                        attemptOpposite[i] = attemptOpposite[i].RemoveAll(option => option.Line[index] != constriant);
                    }

                    if (attemptOpposite.All(opts => opts.Length > 0))
                    {
                        var oppositeFinal = attemptOpposite.ToImmutableArray();
                        var optionFinal = optionAxis.Select((regular, idx) => idx == index ? ImmutableArray.Create(attempt) : regular).ToImmutableArray();

                        var newRoot = (dir == Direction.Horizontal) ?
                            new GridState(
                                Down: oppositeFinal,
                                Across: optionFinal
                                ) :
                            new GridState(
                                Down: optionFinal,
                                Across: oppositeFinal
                                );

                        foreach (var final in AllPossibleGrids(newRoot)) yield return final;
                    }
                }
            }
        }

        static bool Changed(GridState before, GridState after)
        {
            for (int i = 0; i < before.Down.Length; ++i)
            {
                if (before.Down[i].Length != after.Down[i].Length) return true;
                if (before.Across[i].Length != after.Across[i].Length) return true;
            }
            return false;
        }


        private static Comparison<T> Reversed<T>(Comparison<T> original)
        {
            return (x, y) => original(y, x);
        }
        private static int BiasedRandom(int max)
        {
            int random = Random.Shared.Next((max * (max - 1)) / 2);

            int threshold = 0;
            for (int i = 0; i < max; ++i)
            {
                threshold += max - (i+1);

                if (random <= threshold) return i;
            }

            throw new Exception($"Unexpected {threshold} < {random} (out of max {max})");
        }

        //private (
        //    ImmutableArray<ImmutableArray<ScoredLine>> down,
        //    ImmutableArray<ImmutableArray<ScoredLine>> across
        //) OneAttempt(ImmutableArray<ImmutableArray<ScoredLine>> down,
        //    ImmutableArray<ImmutableArray<ScoredLine>> across)
        //{
        //    var dir = new[] { Direction.Horizontal, Direction.Vertical }[Random.Shared.Next(2)];

        //    var line = Random.Shared.Next(0, gridSize);

        //    var options = (dir == Direction.Horizontal ? across : down)[line];

        //    var attempt = options[BiasedRandom(options.Length)];
        //    var opposite = dir == Direction.Horizontal ? down : across;
        //    var newOpposite = ImmutableArray.CreateBuilder<ImmutableArray<ScoredLine>>(opposite.Length);

        //    for (int i = 0; i < gridSize; i++)
        //    {
        //        // WLOG say we dir is Horizontal, and opopsite is Vertical.
        //        // we have:
        //        //
        //        // W O R D
        //        // _ _ _ _
        //        // _ _ _ _
        //        // _ _ _ _
        //        //
        //        // Then go over each COL (i), filtering s.t. possible lines
        //        // only include cases where col[i]'s |attempt|th character == attempt[i].
        //        var constriant = attempt.Line[i];
        //        var currentOptions = opposite[i]
        //            .Where(option => option.Line[line] == constriant)
        //            .ToImmutableArray();

        //        if (currentOptions.Length == 0)
        //        {
        //            // This is not acceptable.
        //            // Fail this attempt.
        //            return dir == Direction.Horizontal
        //                ? (down, across. )
        //                //? (down, across.RemoveAt(line))
        //                //: (down.RemoveAt(line), across);
        //        }

        //        opposite[i] = currentOptions;
        //    }
        //}

        

        public char?[,] GenerateGrid()
        {
            var usedWords = new HashSet<string>();

            char?[,] grid = new char?[gridSize, gridSize];

            if (gridSize == 5)
            {
                var possibleLines = AllPossibleLines(gridSize)
                    .GroupBy(l => l.Line).Select(grp => grp.First())
                    .OrderBy(l => Random.Shared.Next(int.MaxValue))
                    .ToImmutableArray()
                    .Sort(Reversed<ScoredLine>((x, y) =>
                {
                    if (x.MaxLength > y.MaxLength) return +1; // x is greater than y.
                    else if (x.MaxLength < y.MaxLength) return -1; // x is less than y.

                    var xValue = x.NumLetters + x.NumLettersOfCommonWords;
                    var yValue = y.NumLetters + y.NumLettersOfCommonWords;
                    return xValue - yValue;
                }));

                GridState state = new(
                    Down: Enumerable.Range(0, gridSize).Select(_ => possibleLines).ToImmutableArray(),
                    Across: Enumerable.Range(0, gridSize).Select(_ => possibleLines).ToImmutableArray()
                );

                int tries = 0;
                Direction direction = Direction.Horizontal;
                while (tries < 4)
                {
                    ++tries;
                    GridState newState = Prefilter(state, direction);
                    if (!Changed(state, newState)) break;

                    state = newState;
                    direction = direction == Direction.Vertical ? Direction.Horizontal : Direction.Vertical;
                }

                foreach (var x in AllPossibleGrids(state))
                {
                    Console.WriteLine(x);
                }

                GenHelper.Fill(grid, 0, Direction.Horizontal, 0, state.Across[0][Random.Shared.Next(state.Across[0].Length)].Line);
                GenHelper.Fill(grid, 1, Direction.Horizontal, 0, state.Across[1][Random.Shared.Next(state.Across[1].Length)].Line);
                GenHelper.Fill(grid, 2, Direction.Horizontal, 0, state.Across[2][Random.Shared.Next(state.Across[2].Length)].Line);
                GenHelper.Fill(grid, 3, Direction.Horizontal, 0, state.Across[3][Random.Shared.Next(state.Across[3].Length)].Line);
                return grid;
            }

            var startDirection = new[] { Direction.Horizontal, Direction.Vertical }[Random.Shared.Next(2)];
            int ODD_OR_EVEN = new int[] { 0, 1 }[Random.Shared.Next(2)];

            for (int q = 0; q < 3; ++q)
            {
                int index = 2 * Random.Shared.Next(gridSize / 2) + ODD_OR_EVEN;
                var props = GenHelper.FindStartPositionAndLength(grid, index, startDirection);

                if (props == null) continue;

                var (start, length, constraints) = props.Value;

                var possibleWords = commonWordsByLength[length]
                    .Where(word => GenHelper.WordsCompatible(word, constraints) && !usedWords.Contains(word)
                    ).ToArray();

                if (possibleWords.Length == 0) continue;

                var chosenWord = possibleWords[Random.Shared.Next(possibleWords.Length)];
                usedWords.Add(chosenWord);

                GenHelper.Fill(grid, index, startDirection, start, chosenWord);

                startDirection = startDirection switch
                {
                    Direction.Horizontal => Direction.Vertical,
                    Direction.Vertical => Direction.Horizontal,
                    _ => Direction.Horizontal,
                };
            }

            for (int q = 0; q < 0; ++q)
            {
                int index = 2 * Random.Shared.Next(gridSize / 2) + ODD_OR_EVEN;
                var props = GenHelper.FindStartPositionAndLength(grid, index, startDirection);
                if (props == null) continue;
                var (start, length, constraints) = props.Value;

                string[]? possibleWords = commonWords
                    .Where(word =>
                    {
                        if (word.Length > length) return false;
                        if (word.Length == length) return true;
                        if (length - word.Length == 2 || length - word.Length == 3) return false;

                        if (word.Length != length)
                        {
                            // Another use case: We never want the block added in [rowOrCol, start+word.Length]
                            // to split the board s.t. a section is 2 characters or less
                            var colOrRow = start + word.Length;
                            var otherDirection = startDirection switch
                            {
                                Direction.Horizontal => Direction.Vertical,
                                _ => Direction.Horizontal
                            };

                            int clearanceFromEnd = GenHelper.ClearanceFromEnd(grid, colOrRow, index, otherDirection);
                            if (clearanceFromEnd == 1 || clearanceFromEnd == 2) return false;

                            int clearanceFromStart = GenHelper.ClearanceFromStart(grid, colOrRow, index, otherDirection);
                            if (clearanceFromStart == 1 || clearanceFromStart == 2) return false;
                        }

                        char? c = GenHelper.AtPosition(grid, index, start + word.Length, startDirection);

                        return (c == null || c == GenHelper.BLOCKED);
                    })
                    .Where(word => GenHelper.WordsCompatible(word, constraints) && !usedWords.Contains(word))
                    .GroupBy(word => word.Length)
                    .OrderByDescending(group => group.Key)
                    .FirstOrDefault()?.ToArray();

                if (possibleWords == null) continue;

                var chosenWord = possibleWords[Random.Shared.Next(possibleWords.Length)];
                usedWords.Add(chosenWord);

                if (chosenWord.Length == length)
                    GenHelper.Fill(grid, index, startDirection, start, chosenWord);
                else
                    GenHelper.Fill(grid, index, startDirection, start, chosenWord + GenHelper.BLOCKED);

                startDirection = startDirection switch
                {
                    Direction.Horizontal => Direction.Vertical,
                    Direction.Vertical => Direction.Horizontal,
                    _ => Direction.Horizontal,
                };
            }

            for (int q = 0; q < 400; ++q)
            {
                int index = Random.Shared.Next(gridSize);

                var props = GenHelper.FindStartPositionAndLength(grid, index, startDirection);
                if (props == null) continue;
                var (start, length, constraints) = props.Value;

                string? word = FindWordWithConstraints(usedWords, startDirection, index, start, length, constraints, grid);
                if (word == null) continue;

                usedWords.Add(word);
                GenHelper.Fill(grid, index, startDirection, start, word);

                startDirection = startDirection switch
                {
                    Direction.Horizontal => Direction.Vertical,
                    Direction.Vertical => Direction.Horizontal,
                    _ => Direction.Horizontal,
                };
            }

            foreach (Direction d in new[] { Direction.Horizontal, Direction.Vertical })
            {
                for (int i = 0; i < gridSize; ++i)
                {
                    (int, int, char?[])? props = null;

                    while (null != (props = GenHelper.FindStartPositionAndLength(grid, i, d)))
                    {
                        var (start, length, constraints) = props.Value;
                        string? word = FindWordWithConstraints(usedWords, d, i, start, length, constraints, grid);

                        if (word == null) word = string.Join("", constraints.Select(c => c == null ? '?' : c));

                        GenHelper.Fill(grid, i, d, start, word);
                    }
                }
            }

            return grid;
        }

        private string? FindWordWithConstraints(HashSet<string> usedWords, Direction direction, int index, int start, int length, char?[] constraints, char?[,] grid)
        {
            return FindWordsWithConstraints(usedWords, direction, index, start, length, constraints, grid)
            .FirstOrDefault();
        }

        public IEnumerable<string> FindWordsWithConstraints(HashSet<string> usedWords, Direction direction, int index, int start, int length, char?[] constraints, char?[,] grid)
        {
            int gridSize = grid.GetLength(0);

            return GenHelper.CompatibleWords(constraints, wordsByLetterPositon, commonAndObscureWords)
            .Where(word => word.Length <= length)
            //.SelectMany<string, (int ogWordLength, string word)>(word =>
            //{
            //    int ogWordLength = word.Length;

            //    if (ogWordLength == length) return new[] { (ogWordLength, word) };
            //    int lengthDiff = length - ogWordLength;

            //    return lengthDiff switch
            //    {
            //        <= 0 => new[] { (ogWordLength, word) },
            //        1 => new[]
            //        {
            //            (ogWordLength, GenHelper.BLOCKED + word),
            //            (ogWordLength, word + GenHelper.BLOCKED),
            //        },
            //        2 => new[]
            //        {
            //            (ogWordLength, GenHelper.BLOCKED + GenHelper.BLOCKED + word),
            //            (ogWordLength, GenHelper.BLOCKED + word + GenHelper.BLOCKED),
            //            (ogWordLength, word + GenHelper.BLOCKED + GenHelper.BLOCKED),
            //        },
            //        3 => new[]
            //        {
            //            (ogWordLength, GenHelper.BLOCKED + GenHelper.BLOCKED + GenHelper.BLOCKED + word),
            //            (ogWordLength, GenHelper.BLOCKED + GenHelper.BLOCKED + word + GenHelper.BLOCKED),
            //            (ogWordLength, GenHelper.BLOCKED + word + GenHelper.BLOCKED + GenHelper.BLOCKED),
            //            (ogWordLength, word + GenHelper.BLOCKED + GenHelper.BLOCKED + GenHelper.BLOCKED),
            //        },
            //        _ => new[]
            //        {
            //            (ogWordLength, new string(' ', lengthDiff - 1) + GenHelper.BLOCKED + word),
            //            (ogWordLength, word + GenHelper.BLOCKED),
            //        },
            //    };
            //})
            .Select<string, (int ogWordLength, string word)>(word =>
            {
                int ogWordLength = word.Length;
                int lengthDifference = length - ogWordLength;

                return lengthDifference switch
                {
                    <= 0 => (ogWordLength, word),
                    >= 1 and <= 3 => (ogWordLength, word + new string(GenHelper.BLOCKED, lengthDifference)),
                    >= 4 => (ogWordLength, word + GenHelper.BLOCKED),
                };
            })
            .Where(wl => {
                string word = wl.word;

                for (int i = 0; i < word.Length; ++i)
                {
                    if (word[i] != GenHelper.BLOCKED) continue;

                    // Any blocks we add need to be at a non-conflicting place.
                    char? c = GenHelper.AtPosition(grid, index, start + i, direction);
                    if (!(c == null || c == GenHelper.BLOCKED)) return false;

                    // We never want the block added in [rowOrCol, start+word.Length]
                    // to split the board s.t. a section is 2 characters or less.
                    var colOrRow = start + i;
                    var otherDirection = direction switch
                    {
                        Direction.Horizontal => Direction.Vertical,
                        _ => Direction.Horizontal
                    };

                    int clearanceFromEnd = GenHelper.ClearanceFromEnd(grid, colOrRow, index, otherDirection);
                    if (clearanceFromEnd == 1 || clearanceFromEnd == 2) return false;

                    int clearanceFromStart = GenHelper.ClearanceFromStart(grid, colOrRow, index, otherDirection);
                    if (clearanceFromStart == 1 || clearanceFromStart == 2) return false;
                }
                return true;
            })
            .OrderByDescending(tp => tp.ogWordLength)
            .Select(tp => tp.word)
            .Where(word =>
            {
                for (int i = 0; i < word.Length; ++i)
                {
                    var oldLine = string.Join(GenHelper.BLOCKED, GridReader.ReadLine(grid, start + i, direction switch { Direction.Horizontal => Direction.Vertical, _ => Direction.Horizontal }));
                    var newLine =
                        (index == 0 ? $"{word[i]}{oldLine.AsSpan()[1..]}" :
                        index == gridSize - 1 ? $"{oldLine.AsSpan()[..^1]}{word[i]}" :
                        $"{oldLine.AsSpan()[..(index - 1)]}{word[i]}{oldLine.AsSpan()[(index + 1)..]}").Split(GenHelper.BLOCKED);

                    foreach (string newLineWord in newLine)
                    {
                        // If the new word we formed is a fully-formed word, it works.
                        // Don't check every possible word.
                        if (newLineWord.Trim().Length == 0) continue;
                        if (commonAndObscureWords.Contains(newLineWord)) continue;
                        var length = newLineWord.Length;

                        if (false == commonAndObscureWords
                            .Where(word => word.Length <= length)
                            // Consider making this less strict: (e. accept diff 2,3)
                            .Where(word =>
                            {
                                var diff = length - word.Length;
                                return diff != 2 && diff != 3;
                            })
                            .Where(word => GenHelper.WordsCompatible(word, newLineWord))
                            .Any()
                        )
                        {
                            return false;
                        }
                    }
                }
                return true;
            }
            );
        }

        public IEnumerable<string> CompatibleWords(char?[] template) => GenHelper.CompatibleWords(template, wordsByLetterPositon, commonAndObscureWords);
    }

    internal static class GenHelper
    {
        public const char BLOCKED = '`';

        public static bool WordsCompatible(string word, string template)
        {
            if (word.Length > template.Length) return false;
            for (int i = 0; i < word.Length; ++i)
            {
                char t = template[i];
                if (t != word[i] && t != ' ') return false;
            }
            return true;
        }
        public static bool WordsCompatible(string word, char?[] template)
        {
            if (word.Length > template.Length) return false;
            for (int i = 0; i < word.Length; ++i)
            {
                char? t = template[i];
                if (t != word[i] && t != null) return false;
            }
            return true;
        }
        public static IEnumerable<string> CompatibleWords(char?[] template, ImmutableArray<ImmutableDictionary<char, ImmutableSortedSet<string>>> wordsByLetterPosition, ImmutableArray<string> allWords)
        {
            ImmutableSortedSet<string>? candidates = null;

            for (int i = 0; i < template.Length; ++i)
            {
                char? t = template[i];
                if (t == null || t == '?') continue;

                var possibleWords = wordsByLetterPosition[i][t.Value];
                if (candidates == null) candidates = possibleWords;
                else candidates = candidates.Intersect(possibleWords);
            }

            if (candidates == null) return allWords;
            return candidates;
        }

        public static (int, int, char?[])? FindStartPositionAndLength(char?[,] grid, int rowOrCol, Direction dir)
        {
            int gridSize = grid.GetLength(0);

            string[] segments = GridReader.ReadLine(grid, rowOrCol, dir);

            int[] viableSegmentIndices = segments
                .Select((segment, index) => (segment, index))
                .Where(tuple => tuple.segment.Contains(' ') && tuple.segment.Length > 2)
                .Select(tuple => tuple.index)
                .ToArray();
            if (viableSegmentIndices.Length == 0) return null;
            int segmentIndex = viableSegmentIndices[Random.Shared.Next(viableSegmentIndices.Length)];

            int segmentStartsAt = 0;
            for (int i = 0; i < segments.Length; segmentStartsAt += 1 + segments[i].Length, ++i)
            {
                var word = segments[i];
                if (i == segmentIndex) return (segmentStartsAt, word.Length, word.Select<char, char?>(c => c == ' ' ? null : c).ToArray());
            }

            return null;
        }

        public static void Fill(char?[,] grid, int rowOrCol, Direction dir, int start, string word)
        {
            if (dir == Direction.Horizontal)
            {
                for (int i = 0; i < word.Length; ++i)
                {
                    if (word[i] == ' ') continue;
                    grid[start + i, rowOrCol] = word[i];
                }
            }
            else
            {
                for (int i = 0; i < word.Length; ++i)
                {
                    if (word[i] == ' ') continue;
                    grid[rowOrCol, start + i] = word[i];
                }
            }
        }

        public static char? AtPosition(char?[,] grid, int rowOrCol, int position, Direction dir)
        {
            if (dir == Direction.Horizontal)
                return grid[position, rowOrCol];
            else return grid[rowOrCol, position];
        }

        public static int ClearanceFromStart(char?[,] grid, int rowOrCol, int posWithinLine, Direction dir)
        {
            int room = 0;
            for (int pos = 0; pos < posWithinLine; ++pos)
            {
                if (BLOCKED == AtPosition(grid, rowOrCol, pos, dir)) room = 0;
                else room += 1;
            }
            return room;
        }

        public static int ClearanceFromEnd(char?[,] grid, int rowOrCol, int posWithinLine, Direction dir)
        {
            int gridSize = grid.GetLength(0);
            int pos = posWithinLine;
            for (; pos < gridSize; ++pos)
            {
                if (BLOCKED == AtPosition(grid, rowOrCol, pos, dir)) break;
            }
            return pos - posWithinLine;
        }
    }

    public static class GridReader
    {
        internal static ImmutableArray<string> COMMON_WORDS =
            Properties.Resources.words.Split(new[] { "\r\n", "\r", "\n" }, StringSplitOptions.None)
            .Concat(Properties.Resources.phrases.Split(new[] { "\r\n", "\r", "\n" }, StringSplitOptions.None))
            .Select(s => s.Trim().Replace(" ", ""))
            .ToImmutableArray();

        internal static ImmutableArray<string> OBSCURE_WORDS =
            Properties.Resources.phrases.Split(new[] { "\r\n", "\r", "\n" }, StringSplitOptions.None)
            .Concat(Properties.Resources.wikipedia.Split(new[] { "\r\n", "\r", "\n" }, StringSplitOptions.None))
            .Concat(Properties.Resources.from_lexems.Split(new[] { "\r\n", "\r", "\n" }, StringSplitOptions.None))
            .Select(s => s.Trim().Replace(" ", ""))
            .ToImmutableArray();

        internal static ImmutableArray<string> ALL_WORDS = COMMON_WORDS.AddRange(OBSCURE_WORDS);

        public static HashSet<string> AllowedWords()
        {
            return new HashSet<string>(ALL_WORDS);
        }

        public static char?[] GetConstraints(char?[,] grid, int rowOrCol, int start, int length, Direction dir)
        {
            var cs = new List<char?>();

            for (int i = 0; i < length; ++i)
            {
                var c = GenHelper.AtPosition(grid, rowOrCol, start + i, dir);
                if (c == GenHelper.BLOCKED) break;
                cs.Add(c);
            }
            return cs.ToArray();
        }

        public static string[] ReadLine(char?[,] grid, int rowOrCol, Direction dir)
        {
            int gridSize = grid.GetLength(0);
            var sb = new StringBuilder();
            if (dir == Direction.Horizontal)
            {
                for (int i = 0; i < gridSize; ++i)
                {
                    sb.Append(grid[i, rowOrCol] switch
                    {
                        null => ' ',
                        char c => c,
                    });
                }
            }
            else
            {
                for (int i = 0; i < gridSize; ++i)
                {
                    sb.Append(grid[rowOrCol, i] switch
                    {
                        null => ' ',
                        char c => c,
                    });
                }
            }

            return sb.ToString().Split(GenHelper.BLOCKED);
        }
    }

    public enum Direction { Horizontal, Vertical }
}