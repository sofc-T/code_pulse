import styles from "./LogInForm.module.css";
import { useState } from "react";
import { ToastContainer, toast } from "react-toastify";
import "react-toastify/dist/ReactToastify.css";
import  { useUpdateEmail, useUpdateToken } from "../../store/AuthContext";
import { Link, useNavigate } from "react-router-dom";

const LogInForm = () => {
  const Notify = (message, type) => {
    if (type === "success") {
      return toast.success(message, {
        position: toast.POSITION.BOTTOM_CENTER,
      });
    } else {
      return toast.error(message, {
        position: toast.POSITION.BOTTOM_CENTER,
      });
    }
  };
  function validateEmail(email) {
    const regex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
    return regex.test(email);
  }
  
  function validatePassword(password) {
    const failedPolicies = [];
    return failedPolicies;
  }
  const navigate = useNavigate();
  const updateEmail = useUpdateEmail();
  const setToken = useUpdateToken();
  const [enteredEmail, setEnteredEmail] = useState("");
  const [emailInputTouched, setEmailInputTouched] = useState(false);
  const emailIsValid = validateEmail(enteredEmail);
  const emailInputIsInvalid = !emailIsValid && emailInputTouched;

  const [enteredPassword, setEnteredPassword] = useState("");
  const [passwordInputTouched, setPasswordInputTouched] = useState(false);
  const passwordIsValid = validatePassword(enteredPassword).length === 0;
  const passwordInputIsInvalid = !passwordIsValid && passwordInputTouched;

  const [isLoading, setisLoading] = useState(false);

  const emailInputChangeHandler = (event) => {
    setEnteredEmail(event.target.value);
  };
  const emailInputBlurHandler = (event) => {
    setEmailInputTouched(true);
  };

  const passwordInputChangeHandler = (event) => {
    setEnteredPassword(event.target.value);
  };
  const passwordInputBlurHandler = (event) => {
    setPasswordInputTouched(true);
  };

  const formSubmissionHandler = (event) => {
    event.preventDefault();

    setEmailInputTouched(true);
    setPasswordInputTouched(true);

    if (!emailIsValid) {
      return;
    }
    if (!passwordIsValid) {
      return;
    }
    setisLoading(true);

    const LogInDto = {
      email: enteredEmail,
      password: enteredPassword,
    };
    const BASE_URL = process.env.REACT_APP_BASE_URL;
    fetch(`${BASE_URL}/auth/login`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify(LogInDto),
    })
      .then(async (response) => {
        if (response.ok) {
          const data = await response.json();
          if (data.token) {
            localStorage.setItem("authToken",data.token);
            localStorage.setItem("email",enteredEmail);
            setToken(localStorage.getItem("authToken"));
            updateEmail(enteredEmail);
          }
          Notify(data.message, "error");
          navigate("/home");
        } else {
          const data = await response.json();
          Notify(data.message, "error");
        }
      })
      .finally(() => {
        setisLoading(false);
      });
    setEnteredEmail("");
    setEmailInputTouched(false);
    setEnteredPassword("");
    setPasswordInputTouched(false);
  };

  const passwordInfo = (
    <ul style={{ paddingLeft: "30px" }}>
      {validatePassword(enteredPassword).map((policy, index) => (
        <li className={styles["error-text"]} key={index}>
          {policy}
        </li>
      ))}
    </ul>
  );

  return (
    <div>
      <div className={styles.login}>
        <div className={styles["form-container"]}>
          <h1>Login</h1>
          <form onSubmit={formSubmissionHandler}>
            <div className={styles.inputContainer}>
              <label htmlFor="email" className={styles.hidden}>
                Email
              </label>
              <input
                type="email"
                id="email"
                placeholder="Email"
                value={enteredEmail}
                onBlur={emailInputBlurHandler}
                onChange={emailInputChangeHandler}
                aria-invalid={emailInputIsInvalid}
                aria-describedby={emailInputIsInvalid ? "emailError" : null}
              />
              {emailInputIsInvalid && (
                <p id="emailError" className={styles["error-text"]}>
                  Please enter a valid email
                </p>
              )}
            </div>
            <div className={styles.inputContainer}>
              <label htmlFor="password" className={styles.hidden}>
                Password
              </label>
              <input
                type="password"
                id="password"
                placeholder="Password"
                value={enteredPassword}
                onBlur={passwordInputBlurHandler}
                onChange={passwordInputChangeHandler}
                aria-invalid={passwordInputIsInvalid}
                aria-describedby={
                  passwordInputIsInvalid ? "passwordError" : null
                }
              />
              {passwordInputIsInvalid && (
                <div id="passwordError" className={styles["error-text"]}>
                  {passwordInfo}
                </div>
              )}
            </div>
            <button
              className={styles["form-submit"]}
              disabled={isLoading}
              type="submit"
            >
              Login
            </button>
          </form>
          <div className={styles.login}>
          <p>Don't have an account? <Link to="/signup">
            <u>Signup</u>
          </Link> </p>
        </div>
        </div>
      </div>
      <ToastContainer />
    </div>
  );
};

export default LogInForm;

