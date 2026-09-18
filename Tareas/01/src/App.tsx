import { useEffect, useState } from 'react'

interface Pokemon {
  name: string
  url: string
  id: number
}

interface PokemonDetail {
  id: number
  name: string
  types: { type: { name: string } }[]
  stats: { stat: { name: string }; base_stat: number }[]
  height: number
  weight: number
  base_experience: number
  moves: { move: { name: string } }[]
  sprites: { other: { 'official-artwork': { front_default: string } } }
}

interface EvolutionItem {
  name: string
}

const typeColors: { [key: string]: string } = {
  normal: 'bg-gray-400',
  fire: 'bg-red-500',
  water: 'bg-blue-500',
  grass: 'bg-green-500',
  electric: 'bg-yellow-400',
  ice: 'bg-cyan-300',
  fighting: 'bg-red-700',
  poison: 'bg-purple-500',
  ground: 'bg-yellow-600',
  flying: 'bg-indigo-400',
  psychic: 'bg-pink-500',
  bug: 'bg-lime-500',
  rock: 'bg-gray-600',
  ghost: 'bg-purple-700',
  dragon: 'bg-indigo-600',
  dark: 'bg-gray-800',
  steel: 'bg-gray-500',
  fairy: 'bg-pink-300'
}

function PokemonDetailModal({ pokemon, evolutionItems, onClose }: { pokemon: PokemonDetail | null; evolutionItems: EvolutionItem[]; onClose: () => void }) {
  if (!pokemon) return null

  const moves = pokemon.moves.slice(0, 8).map(m => m.move.name)

  return (
    <div className="modal-overlay fixed inset-0 flex items-center justify-center z-50 p-4">
      <div className="modal-content max-w-2xl w-full max-h-[90vh] overflow-y-auto">
        <button
          onClick={onClose}
          className="close-btn absolute top-4 right-4 text-white rounded-full w-8 h-8 flex items-center justify-center"
        >
          ✕
        </button>

        <div className="modal-header p-8 text-center">
          <img
            src={`https://raw.githubusercontent.com/PokeAPI/sprites/master/sprites/pokemon/other/official-artwork/${pokemon.id}.png`}
            alt={pokemon.name}
            className="w-48 h-48 mx-auto mb-4 object-contain"
          />
          <h2 className="text-4xl font-bold text-white capitalize mb-2">
            {pokemon.name}
          </h2>
          <p className="text-slate-400 text-lg mb-4">
            #{String(pokemon.id).padStart(3, '0')}
          </p>

          <div className="flex gap-2 justify-center mb-6 flex-wrap">
            {pokemon.types.map((t) => (
              <span
                key={t.type.name}
                className={`type-badge ${typeColors[t.type.name] || 'bg-gray-500'} text-white px-3 py-1 rounded-full capitalize`}
              >
                {t.type.name}
              </span>
            ))}
          </div>

          <div className="grid grid-cols-3 gap-4 mb-6">
            <div className="info-card p-4">
              <p className="text-slate-300 text-sm mb-1">Experiencia Base</p>
              <p className="text-cyan-300 text-lg font-semibold">
                {pokemon.base_experience}
              </p>
            </div>
            <div className="info-card p-4">
              <p className="text-slate-300 text-sm mb-1">Altura</p>
              <p className="text-cyan-300 text-lg font-semibold">
                {(pokemon.height / 10).toFixed(2)} m
              </p>
            </div>
            <div className="info-card p-4">
              <p className="text-slate-300 text-sm mb-1">Peso</p>
              <p className="text-cyan-300 text-lg font-semibold">
                {(pokemon.weight / 10).toFixed(2)} kg
              </p>
            </div>
          </div>

          {evolutionItems.length > 0 && (
            <div className="mb-6">
              <h3 className="text-xl font-bold text-cyan-300 text-left mb-3">
                Objetos para Evolución
              </h3>
              <div className="grid grid-cols-2 gap-2">
                {evolutionItems.map((item, i) => (
                  <div key={i} className="evolution-item px-3 py-2 rounded text-sm capitalize">
                    ⚡ {item.name.replace('-', ' ')}
                  </div>
                ))}
              </div>
            </div>
          )}

          <div className="mb-6">
            <h3 className="text-xl font-bold text-white text-left mb-4">
              Estadísticas
            </h3>
            <div className="space-y-3">
              {pokemon.stats.map((stat) => (
                <div key={stat.stat.name}>
                  <div className="flex justify-between mb-1">
                    <span className="text-cyan-300 capitalize text-sm font-semibold">
                      {stat.stat.name}
                    </span>
                    <span className="text-cyan-400 font-bold">
                      {stat.base_stat}
                    </span>
                  </div>
                  <div className="stat-bar-container h-2 rounded-full">
                    <div
                      className="stat-bar h-full rounded-full"
                      style={{
                        width: `${Math.min((stat.base_stat / 150) * 100, 100)}%`
                      }}
                    />
                  </div>
                </div>
              ))}
            </div>
          </div>

          <div>
            <h3 className="text-xl font-bold text-white text-left mb-4">
              Ataques (primeros 8)
            </h3>
            <div className="grid grid-cols-2 gap-2">
              {moves.map((move, i) => (
                <div key={i} className="bg-slate-700 px-3 py-2 rounded text-slate-300 text-sm capitalize">
                  {move}
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}

function App() {
  const [pokemon, setPokemon] = useState<Pokemon[]>([])
  const [loading, setLoading] = useState(true)
  const [selectedPokemon, setSelectedPokemon] = useState<PokemonDetail | null>(null)
  const [evolutionItems, setEvolutionItems] = useState<EvolutionItem[]>([])
  const [detailLoading, setDetailLoading] = useState(false)

  useEffect(() => {
    const fetchPokemon = async () => {
      try {
        const res = await fetch('https://pokeapi.co/api/v2/pokemon?limit=151')
        const data = await res.json()

        const pokemonList = data.results.map((p: { name: string; url: string }) => {
          const id = p.url.split('/').filter((x: string) => x).pop()
          return {
            name: p.name,
            url: p.url,
            id: parseInt(id as string)
          }
        })

        setPokemon(pokemonList)
        setLoading(false)
      } catch (error) {
        console.error('Error fetching pokemon:', error)
        setLoading(false)
      }
    }

    fetchPokemon()
  }, [])

  const handlePokemonClick = async (p: Pokemon) => {
    setDetailLoading(true)
    try {
      const res = await fetch(`https://pokeapi.co/api/v2/pokemon/${p.id}`)
      const data = await res.json()
      setSelectedPokemon(data)

      const items: EvolutionItem[] = []

      try {
        const speciesRes = await fetch(`https://pokeapi.co/api/v2/pokemon-species/${p.id}`)
        const speciesData = await speciesRes.json()

        if (speciesData.evolution_chain?.url) {
          const evolutionRes = await fetch(speciesData.evolution_chain.url)
          const evolutionData = await evolutionRes.json()

          const checkEvolution = (chain: any) => {
            if (chain.evolves_to && chain.evolves_to.length > 0) {
              chain.evolves_to.forEach((evo: any) => {
                if (evo.evolution_details && evo.evolution_details.length > 0) {
                  evo.evolution_details.forEach((detail: any) => {
                    if (detail.item) {
                      const itemName = detail.item.name
                      if (!items.find(i => i.name === itemName)) {
                        items.push({ name: itemName })
                      }
                    }
                  })
                }
                checkEvolution(evo)
              })
            }
          }

          checkEvolution(evolutionData.chain)
        }
      } catch (error) {
        console.error('Error fetching evolution items:', error)
      }

      setEvolutionItems(items)
    } catch (error) {
      console.error('Error fetching pokemon detail:', error)
    } finally {
      setDetailLoading(false)
    }
  }

  return (
    <div className="min-h-screen py-8 px-4">
      <div className="max-w-7xl mx-auto">
        <h1 className="text-6xl font-black mb-2 text-center">
          POKÉDEX
        </h1>
        <p className="text-center text-cyan-300 mb-8 font-semibold tracking-wider">
          ⚡ PRIMEROS 151 POKÉMON - CLICK PARA DETALLES ⚡
        </p>

        {loading ? (
          <div className="flex justify-center items-center h-64">
            <p className="text-cyan-300 text-lg font-semibold">▮▯ CARGANDO POKÉMON ▯▮</p>
          </div>
        ) : (
          <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 gap-4">
            {pokemon.map((p) => (
              <div
                key={p.id}
                onClick={() => handlePokemonClick(p)}
                className="pokemon-card p-4 text-center cursor-pointer"
              >
                <img
                  src={`https://raw.githubusercontent.com/PokeAPI/sprites/master/sprites/pokemon/other/official-artwork/${p.id}.png`}
                  alt={p.name}
                  className="w-32 h-32 mx-auto mb-3 object-contain"
                />
                <p className="text-cyan-300 font-bold capitalize text-lg">
                  {p.name}
                </p>
                <p className="text-purple-400 text-sm font-semibold">
                  #{String(p.id).padStart(3, '0')}
                </p>
              </div>
            ))}
          </div>
        )}
      </div>

      {detailLoading && (
        <div className="modal-overlay fixed inset-0 flex items-center justify-center z-50">
          <p className="text-cyan-300 text-lg font-semibold">▮▯ CARGANDO DETALLES ▯▮</p>
        </div>
      )}

      <PokemonDetailModal
        pokemon={selectedPokemon}
        evolutionItems={evolutionItems}
        onClose={() => setSelectedPokemon(null)}
      />
    </div>
  )
}

export default App
