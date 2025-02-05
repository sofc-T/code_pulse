import { useCallback, useEffect, useState } from "react";
import "highlight.js/styles/github-dark.css";
import hljs from "highlight.js";
import Quill from "quill";
import "quill/dist/quill.snow.css";
import { useNavigate, useParams } from "react-router-dom";
import EditorHeader from "../../components/EditorHeader/EditorHeader";
import {
  useIsOpen,
  useToggleOffline,
  useToggleOpen,
  useUpdateTitle,
} from "../../store/UIContext";
import Modal from "../../components/Modal/Modal";
import EditDocDetail from "../EditDocDetail/EditDocDetail";
import {
  useAuthor,
  useEmail,
  useToken,
  useUpdateAuthor,
} from "../../store/AuthContext";

const TOOLBAR_OPTIONS = [
  ["code-block"],
  [{ header: [false] }] 
];

const LANGUAGES = [
  "javascript",
  "typescript",
  "python",
  "java",
  "cpp",
  "ruby",
  "go",
  "rust",
  "sql",
  "html",
  "css"
];

export default function TextEditor() {
  const navigate = useNavigate();
  const { id: documentId } = useParams();
  const [socket, setSocket] = useState();
  const [quill, setQuill] = useState();
  const [initialData, setInitialData] = useState();
  const [isReadOnly, setIsReadOnly] = useState(false);
  const [suggestions, setSuggestions] = useState([]);
  const [showSuggestions, setShowSuggestions] = useState(false);
  const [cursorPosition, setCursorPosition] = useState({ top: 0, left: 0 });

  const email = useEmail();
  const setIsOffline = useToggleOffline();
  const token = useToken();
  const author = useAuthor();
  const setAuthor = useUpdateAuthor();
  const setTitle = useUpdateTitle();
  const isOpen = useIsOpen();
  const onClose = useToggleOpen();


  useEffect(() => {
    const fetchDocument = async () => {
      const BASE_URL = process.env.REACT_APP_BASE_URL;
      try {
        const response = await fetch(
          `${BASE_URL}/documents/getone/${documentId}`,
          {
            method: "POST",
            headers: {
              "Content-Type": "application/json",
              Authorization: "Bearer " + token,
            },
          }
        );
        let text = await response.text(); 
        if (text.endsWith("null")) {
          text = text.slice(0, -4); 
        }
        const cleanText = text.trim(); 
        console.log(cleanText)
        let documentData;
  
        try {
          documentData = JSON.parse(cleanText);
        } catch (parseError) {
          console.error("Error parsing JSON:", parseError);
          return;
        }
  
        setInitialData(documentData.data);
        setTitle(documentData.title);
        setAuthor(documentData.author);
        localStorage.setItem("readAccess", documentData.readAccess);
        localStorage.setItem("writeAccess", documentData.writeAccess);
  
      } catch (error) {
        console.error("Error fetching document:", error.message);
      }
    };
  
    fetchDocument();
  }, [documentId, setTitle, token, setAuthor]);
  
  useEffect(() => {
    if (quill == null || initialData == null || author == null) return;

    quill.setContents(initialData.ops, "api");
    const writeAccess = localStorage.getItem("writeAccess")?.split(",");
    console.log(writeAccess?.includes(email))
    console.log(writeAccess)
    console.log(email)
    if (writeAccess?.includes(email)) {
      quill.enable();
      setIsReadOnly(false);
    } else {
      quill.disable()
      setIsReadOnly(true);
    }
  }, [quill, initialData, email, author]);

  useEffect(() => {
    if (quill == null || documentId == null) return;
    const BASE_URL = process.env.REACT_APP_BASE_URL.slice(7);
    const socketa = new WebSocket(
      `ws://${BASE_URL}/documents/handler?document_id=${documentId}&token=${token}`
    );
    setSocket(socketa);

    return () => {
      socketa.close();
    };
  }, [quill, documentId, token]);

  useEffect(() => {
    if (socket == null || quill == null) return;

    socket.onopen = () => {
      console.log("connected");
      quill.enable();
      setIsOffline(false);
    };

    socket.onmessage = (e) => {
      quill.updateContents(JSON.parse(e.data));
    };

    socket.onerror = (e) => {
      console.log("Error from message", e.message);
    };

    socket.onclose = (e) => {
      console.log(e.code, e.reason);
      if (socket.readyState === WebSocket.CLOSED) {
        quill.disable();
        setIsOffline(true);
      }
    };

    quill.on("text-change", (delta, oldDelta, source) => {
      if (source !== "user") return;
      const message = {
        data: quill.getContents(),
        change: delta,
      };
      socket.send(JSON.stringify(message));
      const range = quill.getSelection();
      if (range) {
        const [line] = quill.getLine(range.index);
        const text = line.domNode.textContent || "";
        
        if (text.startsWith("```")) {
          const query = text.slice(3).toLowerCase();
          const filtered = LANGUAGES.filter(lang => 
            lang.toLowerCase().startsWith(query)
          );
          
          setSuggestions(filtered);
          setShowSuggestions(filtered.length > 0);
          
          const bounds = quill.getBounds(range.index);
          setCursorPosition({
            top: bounds.top + 20,
            left: bounds.left
          });
        } else {
          setShowSuggestions(false);
        }
      }
    });

    return () => {
      socket.close();
    };
  }, [socket, quill]);

  const selectLanguage = (language) => {
    if (!quill) return;
    
    const range = quill.getSelection();
    if (range) {
      const [line] = quill.getLine(range.index);
      const index = quill.getIndex(line);
      quill.deleteText(index, line.length());
      quill.insertText(index, `\`\`\`${language}\n`);
      quill.setSelection(index + language.length + 4, 0);
    }
    setShowSuggestions(false);
  };

  const wrapperRef = useCallback((wrapper) => {
    if (wrapper == null) return;

    wrapper.innerHTML = "";
    const editor = document.createElement("div");
    
    // Add line numbers container
    const lineNumbers = document.createElement("div");
    // lineNumbers.className = "line-numbers";
    // wrapper.appendChild(lineNumbers);
    
    wrapper.appendChild(editor);
    
    const q = new Quill(editor, {
      theme: "snow",
      modules: {
        toolbar: TOOLBAR_OPTIONS,
        syntax: {
          highlight: (text) => hljs.highlightAuto(text).value,
        },
      },
      formats: ["code-block"],
    });

    // Force code-block format on initialization
    q.formatLine(0, q.getLength(), "code-block", true);

    // Initial line numbers
    const updateLineNumbers = () => {
      const lines = q.getLines(0, q.getLength());
      lineNumbers.innerHTML = lines
        .map((_, i) => `<div class="line-number">${i + 1}</div>`)
        .join("");
    };
    updateLineNumbers();

    // Customize code block rendering
    q.on("text-change", () => {
      updateLineNumbers();
      document.querySelectorAll("pre code").forEach((block) => {
        hljs.highlightElement(block);
      });
    });

    q.disable();
    setQuill(q);
  }, []);

  useEffect(() => {
    if (token === null) {
      navigate("/login");
    }
  }, [token, navigate]);

  return (
    <div className="editor-wrapper">
      <Modal isOpen={isOpen} onClose={onClose} childern={<EditDocDetail />} />
      <EditorHeader isReadOnly={isReadOnly} />
      <div className="editor-content">
        <div className="container" ref={wrapperRef}></div>
      </div>
    </div>
  );
}