import { BrowserRouter, Routes, Route, Navigate } from "react-router-dom";
import Login from "./pages/Login";
import Register from "./pages/Register";
import WorkspacePicker from "./pages/WorkspacePicker";
import CreateWorkspace from "./pages/CreateWorkspace";
import WorkspaceView from "./pages/WorkspaceView";
import AcceptInvite from "./pages/AcceptInvite";
import "./App.css";

function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<WorkspacePicker />} />
        <Route path="/login" element={<Login />} />
        <Route path="/register" element={<Register />} />
        <Route path="/create-workspace" element={<CreateWorkspace />} />
        <Route path="/w/:workspaceId" element={<WorkspaceView />} />
        <Route path="/invite/:token" element={<AcceptInvite />} />
        <Route path="*" element={<Navigate to="/" />} />
      </Routes>
    </BrowserRouter>
  );
}

export default App;
