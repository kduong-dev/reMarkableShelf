import { NavLink, Route, Routes } from 'react-router-dom'
import { Library } from './pages/Library'
import { BookDetail } from './pages/BookDetail'
import { Sources } from './pages/Sources'
import { Sync } from './pages/Sync'

function App() {
  return (
    <>
      <header className="app-header">
        <h1 className="app-title">reMarkable Shelf</h1>
        <nav className="app-nav">
          <NavLink to="/" end>
            Library
          </NavLink>
          <NavLink to="/sync">Sync</NavLink>
          <NavLink to="/sources">Sources</NavLink>
        </nav>
      </header>
      <main>
        <Routes>
          <Route path="/" element={<Library />} />
          <Route path="/books/:id" element={<BookDetail />} />
          <Route path="/sync" element={<Sync />} />
          <Route path="/sources" element={<Sources />} />
        </Routes>
      </main>
    </>
  )
}

export default App
