using Crossword;
using System.Collections.Frozen;
using System.Collections.Immutable;
using static WordGenLib.Generator;

namespace WordGenLib
{
    public record class FinalGrid(ImmutableArray<string> Across)
    {
        public string Repr => string.Join('\n', Across);
    }

    public class CharSet(char min, char max)
    {
        private readonly bool[] _available = new bool[1 + (max - min)];
        private readonly char _min = min;
        private int _ct;

        public CharSet() : this('`', 'z') { }

        public void Add(char c)
        {
            if (!_available[c - _min])
            {
                _ct += 1;
                _available[c - _min] = true;
            }
        }

        public void AddAll(CharSet other)
        {
            if (other.IsFull && !IsFull)
            {
                Array.Fill(_available, true);
                _ct = _available.Length;
                return;
            }

            for (char i = other._min; i < other._min + other._ct; ++i)
            {
                if (IsFull) return;
                Add(i);
            }
        }

        public bool Contains(char c)
        {
            return _available[c - _min];
        }

        public bool IsFull => _ct == _available.Length;
    }
    public class Grid2D<T>(int size) where T : notnull, new()
    {
        private readonly T?[,] _values = new T?[size, size];

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

        public T this[int x, int y]
        {
            get
            {
                var v = _values[x, y];
                System.Diagnostics.Debug.Assert(v != null);
                return v;
            }
        }

    }

    internal class LazyGenerator
    {
        public IPossibleLines PossibleLines => possibleLines.Value;

        private readonly Lazy<IPossibleLines> possibleLines;
        private readonly int gridSize;
        private readonly Lazy<Dictionary<int, ImmutableArray<string>>> commonWordsByLength;
        private readonly Lazy<Dictionary<int, ImmutableArray<string>>> obscureWordsByLength;

        internal LazyGenerator(int girdSize, ImmutableArray<string> trimmedCommon, ImmutableArray<string> trimmedObscure, FrozenSet<string> excludeWords)
        {
            this.gridSize = girdSize;
            commonWordsByLength = new(() =>
            {
                var cbl = trimmedCommon
    .Except(excludeWords)
    .GroupBy(w => w.Length)
    .ToDictionary(g => g.Key, g => g.ToImmutableArray());

                for (int i = 3; i <= gridSize; ++i)
                {
                    if (!cbl.ContainsKey(i)) cbl[i] = [];
                }

                return cbl;
            });
            obscureWordsByLength = new(() =>
            {
                var obl = trimmedObscure
    .Except(excludeWords)
    .Except(trimmedCommon)
    .GroupBy(w => w.Length)
    .ToDictionary(g => g.Key, g => g.ToImmutableArray());

                for (int i = 3; i <= gridSize; ++i)
                { if (!obl.ContainsKey(i)) obl[i] = []; }

                return obl;
            });
            possibleLines = new(() => AllPossibleLines(gridSize));
        }

        public WordInfo LookupWord(string word)
        {
            if (commonWordsByLength.Value.TryGetValue(word.Length, out ImmutableArray<string> c) && c.Contains(word))
            {
                return new WordInfo(WordInfo.InclusionType.COMMON);
            }

            if (obscureWordsByLength.Value.TryGetValue(word.Length, out ImmutableArray<string> o) && o.Contains(word))
            {
                return new WordInfo(WordInfo.InclusionType.OBSCURE);
            }

            return new WordInfo(WordInfo.InclusionType.NOT_INCLUDED);
        }

        private IPossibleLines AllPossibleLines(int maxLength)
        {
            return AllPossibleLines(maxLength, [], new DontPrune());
        }
        private IPossibleLines AllPossibleLines(int maxLength, Dictionary<int, IPossibleLines> memo, IPruneStrategy prune)
        {
            if (maxLength > gridSize) throw new ArgumentException($"{nameof(maxLength)} ({maxLength}) cannot be greater than {nameof(gridSize)} {gridSize}");
            if (maxLength < 3) return Impossible.Instance(maxLength);

            if (memo.TryGetValue(maxLength, out var result))
            {
                return result;
            }
            var pruneState = new IPruneStrategy.PruneStepState(maxLength);

            var compoundBuilder = ImmutableArray.CreateBuilder<IPossibleLines>();

            {
                var possibleWords = new Words(
                    Preferred:
                        commonWordsByLength.Value[maxLength]
                            .OrderBy(_ => Random.Shared.Next(int.MaxValue))
                            .Where(_ => prune.ShouldKeepCommonWord(pruneState))
                            .ToImmutableArray(),
                    Obscure:
                        obscureWordsByLength.Value[maxLength]
                            .OrderBy(_ => Random.Shared.Next(int.MaxValue))
                            .Where(_ => prune.ShouldKeepObscureWord(pruneState))
                            .ToImmutableArray()
                    );
                compoundBuilder.Add(possibleWords);
            }

            // recurse into all combination of [ANYTHING]*[ANYTHING]
            //
            // For length 10:
            // 0 1 2 3 4 5 6 7 8 9
            // _ _ _ _ _ _ _ _ _ _
            //       ^     ^
            // Blockage can be anywhere etween idx 3 and len-4 (inclusive).
            if (maxLength >= 7)
            {
                compoundBuilder.AddRange(
                    Enumerable.Range(start: 3, count: maxLength - 6)
                        .Select(i =>
                        {
                            int firstLength = i;  // Always >= 3.
                            int secondLength = maxLength - (i + 1);  // Always >= 3.

                            return (firstLength, secondLength);
                        })
                        .OrderBy(_ => Random.Shared.Next(int.MaxValue))
                        .Select(l => new BlockBetween(
                            AllPossibleLines(l.firstLength, memo, prune),
                            AllPossibleLines(l.secondLength, memo, prune)
                            ))
                    );
            }

            // recurse into *[ANYTHING], and [ANYTHING]*
            {
                var smaller = AllPossibleLines(maxLength - 1, memo, prune);
                if (smaller is not Impossible)
                {
                    compoundBuilder.AddRange(
                        new IPossibleLines[] {
                        new BlockBefore(smaller),
                            new BlockAfter(smaller)
                        }.OrderBy(_ => Random.Shared.Next(int.MaxValue))
                    );
                }
            }

            var options = compoundBuilder switch
            {
                { Count: 0 } => Impossible.Instance(maxLength),
                { Count: 1 } => compoundBuilder[0],
                _ => new Compound(compoundBuilder.ToImmutable()),
            };

            memo.Add(maxLength, options);
            return options;
        }

        public IPossibleLines CompatibleLinesNoExtraBlocks(string template)
        {
            var trimStart = template.TrimStart(Constants.BLOCKED);
            var startBlocks = template.Length - trimStart.Length;
            var trimEnd = trimStart.TrimEnd(Constants.BLOCKED);
            var endBlocks = trimStart.Length - trimEnd.Length;

            IPossibleLines Wrap(IPossibleLines inner)
            {
                for (; startBlocks > 0; startBlocks--)
                {
                    inner = new BlockBefore(inner);
                }
                for (; endBlocks > 0; endBlocks--)
                {
                    inner = new BlockAfter(inner);
                }
                return inner;
            }

            var indexOfBlocked = trimEnd.IndexOf(Constants.BLOCKED);
            if (indexOfBlocked == -1)
            {
                IPossibleLines words = new Words(
                    Preferred:
                        commonWordsByLength.Value[trimEnd.Length]
                            .OrderBy(_ => Random.Shared.Next(int.MaxValue))
                            .ToImmutableArray(),
                    Obscure:
                        obscureWordsByLength.Value[trimEnd.Length]
                            .OrderBy(_ => Random.Shared.Next(int.MaxValue))
                            .ToImmutableArray()
                    );

                for (int i = 0; i < trimEnd.Length; i++)
                {
                    if (trimEnd[i] == ' ') continue;
                    words = words.Filter(trimEnd[i], i);
                }

                return Wrap(words);
            }
            var firstSegment = trimEnd[..indexOfBlocked];
            var secondSegment = trimEnd[(1 + indexOfBlocked)..];

            return Wrap(new BlockBetween(
                First: CompatibleLinesNoExtraBlocks(firstSegment),
                Second: CompatibleLinesNoExtraBlocks(secondSegment)
            ));
        }

        interface IPruneStrategy
        {
            record class PruneStepState(int MaxLength) { }
            bool ShouldKeepCommonWord(PruneStepState state);
            bool ShouldKeepObscureWord(PruneStepState state);
        }
        class DontPrune : IPruneStrategy
        {
            public bool ShouldKeepCommonWord(IPruneStrategy.PruneStepState _) => true;
            public bool ShouldKeepObscureWord(IPruneStrategy.PruneStepState _) => true;
        }
        class PruneRoots : IPruneStrategy
        {
            public bool ShouldKeepCommonWord(IPruneStrategy.PruneStepState s) => Random.Shared.Next(s.MaxLength) > 1;
            public bool ShouldKeepObscureWord(IPruneStrategy.PruneStepState s) => Random.Shared.Next(s.MaxLength) > Math.Min(2, s.MaxLength - 2);
        }
        class PruneAggressive : IPruneStrategy
        {
            public bool ShouldKeepCommonWord(IPruneStrategy.PruneStepState s) => Random.Shared.Next(s.MaxLength) > 1;
            public bool ShouldKeepObscureWord(IPruneStrategy.PruneStepState s) => Random.Shared.Next(s.MaxLength) > Math.Min(3, s.MaxLength - 2);
        }
    }

    public class Generator
    {
        private readonly int gridSize;

        public int GridSize => gridSize;
        public FrozenSet<string> AllowedWords { get; }
        public FrozenSet<string> ExcludedWords { get; }

        private readonly LazyGenerator lazyGenerator;

        public static Generator Create(
            int gridSize,
            ImmutableArray<string>? commonWords = null,
            ImmutableArray<string>? obscureWords = null
        )
        {
            return new(
                gridSize,
                commonWords ?? GridReader.COMMON_WORDS_DEFAULT,
                obscureWords ?? GridReader.OBSCURE_WORDS_DEFAULT,
                GridReader.EXCLUDE_WORDS
                );
        }

        internal Generator(int gridSize, ImmutableArray<string> commonWords, ImmutableArray<string> obscureWords, FrozenSet<string> excludeWords)
        {
            this.gridSize = gridSize;

            var trimmedCommon = commonWords
                .RemoveAll(s => s.Length <= 2 || s.Length > gridSize);
            var trimmedObscure = obscureWords
                .RemoveAll(s => s.Length <= 2 || s.Length > gridSize);

            AllowedWords = trimmedCommon.Concat(trimmedObscure).ToFrozenSet();
            ExcludedWords = excludeWords.ToFrozenSet();

            lazyGenerator = new LazyGenerator(this.gridSize, trimmedCommon, trimmedObscure, excludeWords);
        }

        public record struct WordInfo(WordInfo.InclusionType Inclusion)
        {
            public enum InclusionType { NOT_INCLUDED, COMMON, OBSCURE, EXCLUDED }
        }

        public WordInfo LookupWord(string word)
        {
            return lazyGenerator.LookupWord(word);
        }

        private record GridState(ImmutableArray<IPossibleLines> Down, ImmutableArray<IPossibleLines> Across)
        {
            public int SideLength => Down.Length;
            public int Area => SideLength * SideLength;

            private static int? GetUndecidedWLOG(ImmutableArray<IPossibleLines> Lines)
            {
                var options = Lines
                    .Select((options, index) => (options, index))
                    .Where(oi => oi.options.MaxPossibilities > 1);
                if (!options.Any()) return null;

                long min = options.Min(oi => oi.options.MaxPossibilities);

                return options
                    .Where(oi => oi.options.MaxPossibilities == min)
                    .Select(oi => (int?)oi.index)
                    .OrderBy(_ => Random.Shared.Next())
                    .FirstOrDefault();
            }

            public int? GetUndecidedDown() => GetUndecidedWLOG(Down);
            public int? GetUndecidedAcross() => GetUndecidedWLOG(Across);
        }



        static GridState Prefilter(GridState state, Direction direction)
        {
            var toFilter = direction == Direction.Horizontal ? state.Across : state.Down;
            var constraint = direction == Direction.Horizontal ? state.Down : state.Across;

            if (toFilter.Any(r => r.MaxPossibilities == 0) || constraint.Any(c => c.MaxPossibilities == 0)) return state;

            // x and y here are abstracted wlog based on toFilter/constraint, not truly
            // connected to Horizontal vs Vertical.
            Grid2D<CharSet> available = new(state.Across.Length);
            for (int x = 0; x < constraint.Length; x++)
            {
                var c = constraint[x];

                for (int y = 0; y < c.NumLetters; ++y)
                {
                    var chars = available.GetOrAddDefault((x, y));
                    c.CharsAt(chars, y);
                }
            }

            var filtered = toFilter.Select((possibles, y) =>
            {
                if (Enumerable.Range(0, toFilter.Length)
                    .All(x => available[x, y].IsFull)) return possibles;

                for (int x = 0; x < possibles.NumLetters; ++x)
                {
                    possibles = possibles.Filter(available[x, y], x);
                }
                return possibles;
            }).ToImmutableArray();

            if (direction == Direction.Horizontal)
            {
                return state with { Across = filtered };
            }
            else return state with { Down = filtered };
        }

        private static IEnumerable<FinalGrid> AllPossibleGrids(GridState root)
        {
            // If we are at a point in our tree some row/column is unfillable, prune this tree.
            if (root.Down.Any(options => options.MaxPossibilities == 0)) yield break;
            if (root.Across.Any(options => options.MaxPossibilities == 0)) yield break;

            int priorNumBlocked = Enumerable.Range(0, root.SideLength)
                .Select(i => root.Down.Count(line => line.DefinitelyBlockedAt(i)))
                .Sum();

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
                if (root.Down.Any(options => options.MaxPossibilities == 0)) yield break;
                if (root.Across.Any(options => options.MaxPossibilities == 0)) yield break;
            }

            // If board is > 35% blocked, it's not worth iterating in it.
            int numDefinitelyBlocked = Enumerable.Range(0, root.SideLength)
                .Select(i => root.Down.Count(line => line.DefinitelyBlockedAt(i)))
                .Sum();

            if (numDefinitelyBlocked > (root.Area * 0.35))
            {
                yield break;
            }

            // If board is entirely divided, s.t. no word spans two "halves" of the
            // board, we want to stop.
            //
            // We already can't have entire blocked lines. But we can have:
            // _ _ _ ` ` `
            // ` ` ` _ _ _
            //
            // This can still be better, e.g. it doesn't account for a "quadrant"
            // being cordoned off.
            if (numDefinitelyBlocked > priorNumBlocked)
            {
                if (IsBoardDefinitelyDivided(root)) yield break;
            }

            int? undecidedDown = root.GetUndecidedDown();
            int? undecidedAcross = root.GetUndecidedAcross();

            if (undecidedDown == null && undecidedAcross == null)
            {
                var down = root.Down.Select(col => col.FirstOrNull()?.Line).ToImmutableArray();
                var across = root.Across.Select(row => row.FirstOrNull()?.Line).ToImmutableArray();

                if (down.Concat(across).Any(item => item == null))
                    yield break;

                if (down.Zip(across).Any(both => both.First == both.Second))
                    yield break;

                yield return new FinalGrid(
                    Across: [.. across.Cast<string>()]
                );
                yield break;
            }

            var possibleGrids = (undecidedDown, undecidedAcross) switch
            {
                (int uD, null) => AllPossibleGrids(root, uD, Direction.Vertical),
                (int uD, int uA) when root.Down[uD].MaxPossibilities < root.Across[uA].MaxPossibilities => AllPossibleGrids(root, uD, Direction.Vertical),
                (null, int uA) => AllPossibleGrids(root, uA, Direction.Horizontal),
                (int uD, int uA) when root.Down[uD].MaxPossibilities >= root.Across[uA].MaxPossibilities => AllPossibleGrids(root, uA, Direction.Horizontal),
                _ => [],
            };
            foreach (var final in possibleGrids) yield return final;
        }

        private static IEnumerable<FinalGrid> AllPossibleGrids(GridState root, int index, Direction dir)
        {
            var optionAxis = (dir == Direction.Horizontal ? root.Across : root.Down);
            var oppositeAxis = (dir == Direction.Horizontal ? root.Down : root.Across);

            // Trim situations where horizontal and vertal words are same.
            for (int i = 0; i < optionAxis.Length; ++i)
            {
                if (optionAxis[i].MaxPossibilities > 1) continue;
                if (oppositeAxis[i].MaxPossibilities > 1) continue;

                var optA = optionAxis[i].FirstOrNull();
                var oppA = oppositeAxis[i].FirstOrNull();
                if (optA == null || oppA == null) yield break;
                if (optA! == oppA!) yield break;
            }

            var options = optionAxis[index];

            // The below loop "makes decisions" and recurses. If we already
            // have one possibility, that means it's already pre-decided.
            if (options.MaxPossibilities <= 1) yield break;

            if (options.MaxPossibilities >= 10)
            {
                do
                {
                    var (choice, remaining) = options.MakeChoice();

                    {
                        var attemptOpposite = oppositeAxis.ToArray();
                        var oppositeFinal = attemptOpposite.ToImmutableArray();
                        var optionFinal = optionAxis
                            .Select((regular, idx) =>
                                idx == index
                                    ? choice
                                    : regular)
                            .ToImmutableArray();

                        if (attemptOpposite.Zip(optionFinal)
                            .Where(ab => ab.First.MaxPossibilities == 1 && ab.Second.MaxPossibilities == 1)
                            .Any(ab =>
                            {
                                var f = ab.First.Iterate().Select(i => i as ConcreteLine?).FirstOrDefault();
                                var s = ab.Second.Iterate().Select(i => i as ConcreteLine?).FirstOrDefault();

                                if (f == s && f != null) return true;
                                return false;
                            }
                            ))
                            yield break;

                        var newRoot = (dir == Direction.Horizontal) ?
                            new GridState(
                                Down: oppositeFinal,
                                Across: optionFinal
                                ) :
                            new GridState(
                                Down: optionFinal,
                                Across: oppositeFinal
                                );

                        if (NumDefiniteBlocks(choice) > NumDefiniteBlocks(options))
                        {
                            if (IsBoardDefinitelyDivided(newRoot)) yield break;
                        }

                        foreach (var final in AllPossibleGrids(newRoot)) yield return final;
                    }

                    options = remaining;
                }
                while (options.MaxPossibilities > 1);

                if (options.MaxPossibilities == 0)
                {
                    yield break;
                }
            }

            foreach (var attempt in options.Iterate())
            {
                var attemptIndividualWords = attempt.Words;
                if (attemptIndividualWords.GroupBy(w => w).Any(g => g.Count() > 1)) yield break;

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

                    attemptOpposite[i] = IPossibleLines.RemoveWordOptions(attemptIndividualWords, attemptOpposite[i]).Filter(constriant, index);
                    if (attemptOpposite[i].MaxPossibilities == 1)
                    {
                        var ao = attemptOpposite[i].FirstOrNull();
                        if (ao == null || ao! == attempt) yield break;
                    }
                }

                if (attemptOpposite.All(opts => opts is not Impossible && opts.MaxPossibilities > 0))
                {
                    var oppositeFinal = attemptOpposite.ToImmutableArray();
                    var optionFinal = optionAxis
                        .Select((regular, idx) =>
                            idx == index
                                ? new Definite(attempt)
                                : IPossibleLines.RemoveWordOptions(attemptIndividualWords, regular))
                        .ToImmutableArray();

                    if (attemptOpposite.Zip(optionFinal)
                        .Where(ab => ab.First.MaxPossibilities <= 1 && ab.Second.MaxPossibilities <= 1)
                        .Any(ab =>
                        {
                            var first = ab.First.FirstOrNull();
                            var second = ab.Second.FirstOrNull();
                            if (first == null || second == null) return true;
                            return first! == second!;
                        }))
                        yield break;

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

        static bool Changed(GridState before, GridState after)
        {
            for (int i = 0; i < before.Down.Length; ++i)
            {
                if (before.Down[i].MaxPossibilities != after.Down[i].MaxPossibilities) return true;
                if (before.Across[i].MaxPossibilities != after.Across[i].MaxPossibilities) return true;
            }
            return false;
        }

        static int NumDefiniteBlocks(IPossibleLines state)
        {
            int acc = 0;
            for (int i = 0; i < state.NumLetters; ++i)
            {
                if (state.DefinitelyBlockedAt(i)) acc += 1;
            }
            return acc;
        }

        static bool IsBoardDefinitelyDivided(GridState state)
        {
            char[,] grid = new char[state.SideLength, state.SideLength];
            int unreachable = state.Area;

            for (int i = 0; i < grid.GetLength(0); ++i)
            {
                for (int j = 0; j < grid.GetLength(1); ++j)
                {
                    if (state.Down[i].DefinitelyBlockedAt(j) || state.Across[j].DefinitelyBlockedAt(i))
                    {
                        grid[i, j] = '`';
                        unreachable--;
                    }
                    else
                    {
                        grid[i, j] = ' ';
                    }
                }
            }

            Queue<(int i, int j)> explore = new();
            for (int i = 0; i < grid.GetLength(0); ++i)
            {
                if (grid[i, 0] == ' ')
                {
                    explore.Enqueue((i, 0));
                    break;
                }
            }

            while (explore.TryDequeue(out var ij))
            {
                (int i, int j) = ij;

                if (grid[i, j] != ' ') continue;
                grid[i, j] = '=';
                unreachable--;

                if ((i - 1) >= 0 && grid[i - 1, j] == ' ') explore.Enqueue((i - 1, j));
                if ((i + 1) < grid.GetLength(0) && grid[i + 1, j] == ' ') explore.Enqueue((i + 1, j));
                if ((j - 1) >= 0 && grid[i, j - 1] == ' ') explore.Enqueue((i, j - 1));
                if ((j + 1) < grid.GetLength(1) && grid[i, j + 1] == ' ') explore.Enqueue((i, j + 1));
            }

            if (unreachable > 0) return true;

            for (int i = 0; i < grid.GetLength(0); ++i)
            {
                int numDim1Blocked = 0;
                int numDim2Blocked = 0;
                for (int j = 0; j < grid.GetLength(1); ++j)
                {
                    if (grid[i, j] == '`') ++numDim1Blocked;
                    if (grid[j, i] == '`') ++numDim2Blocked;
                }

                if (numDim1Blocked == grid.GetLength(1) || numDim2Blocked == grid.GetLength(0)) return true;
            }

            return false;
        }

        public IEnumerable<FinalGrid> PossibleGrids()
        {
            GridState state = new(
                Down: [.. Enumerable.Range(0, gridSize).Select(_ => lazyGenerator.PossibleLines)],
                Across: [.. Enumerable.Range(0, gridSize).Select(_ => lazyGenerator.PossibleLines)]
            );
            return AllPossibleGrids(state).DistinctBy(x => x.Repr);
        }

        public IEnumerable<FinalGrid> PossibleGridWithConstraints(char[,] constraints, bool addExtraBlocks)
        {
            ArgumentNullException.ThrowIfNull(constraints);
            if (constraints.GetLength(0) != gridSize || constraints.GetLength(1) != gridSize) throw new ArgumentException($"{nameof(constraints)} should have size {GridSize} x {GridSize}");

            string[] acrossTemplates = [.. Enumerable.Range(0, gridSize).Select(y => string.Join("", Enumerable.Range(0, gridSize).Select(x => constraints[x, y])))];
            string[] downTemplates = [.. Enumerable.Range(0, gridSize).Select(x => string.Join("", Enumerable.Range(0, gridSize).Select(y => constraints[x, y])))];

            GridState state = new(
                Down: [.. Enumerable.Range(0, gridSize).Select(i => CompatibleLines(downTemplates[i], addExtraBlocks))],
                Across: [.. Enumerable.Range(0, gridSize).Select(i => CompatibleLines(acrossTemplates[i], addExtraBlocks))]
            );
            if (IsBoardDefinitelyDivided(state)) return [];

            return AllPossibleGrids(state).DistinctBy(x => x.Repr);
        }

        private IPossibleLines CompatibleLines(string template, bool addExtraBlocks)
        {
            if (addExtraBlocks) return CompatibleLinesExtraBlocks(template);
            else return lazyGenerator.CompatibleLinesNoExtraBlocks(template);
        }

        private IPossibleLines CompatibleLinesExtraBlocks(string template)
        {
            if (template.All(x => x == ' ')) return lazyGenerator.PossibleLines;
            if (template.All(x => x != ' ')) return new Definite(new(
                template,
                [.. template.Split(Constants.BLOCKED).Where(w => w.Length > 0)]
                ));

            var lines = lazyGenerator.PossibleLines;
            for (int i = 0; i < template.Length; i++)
            {
                if (template[i] == ' ') continue;
                lines = lines.Filter(template[i], i);
            }
            return lines;
        }


    }

    internal static class GenHelper
    {
        public static void FillRow(char?[,] grid, int row, string word)
        {
            for (int i = 0; i < word.Length; ++i)
            {
                if (word[i] == ' ') continue;
                grid[i, row] = word[i];
            }
        }
    }

    public static class GridReader
    {
        public static ImmutableArray<string> COMMON_WORDS_DEFAULT => WordLoader.GetRepository().Regular.Common;
        public static ImmutableArray<string> OBSCURE_WORDS_DEFAULT => WordLoader.GetRepository().Regular.Obscure;

        public static readonly FrozenSet<string> EXCLUDE_WORDS =
            Properties.Resources.overused.Split(["\r\n", "\r", "\n"], StringSplitOptions.RemoveEmptyEntries)
            .Select(s => s.Trim().Replace(" ", ""))
            .ToFrozenSet();

        public static string[] AcrossWords(FinalGrid grid)
        {
            return [.. grid.Across.SelectMany(s => s.Split(Constants.BLOCKED)).Where(s => s.Length > 0)];
        }
    }

    public enum Direction { Horizontal, Vertical }
}